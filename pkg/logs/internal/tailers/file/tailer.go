// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package file

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync/atomic"
	"time"

	coreConfig "github.com/DataDog/datadog-agent/pkg/config"
	"github.com/DataDog/datadog-agent/pkg/util/containers"
	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/DataDog/datadog-agent/pkg/logs/internal/decoder"
	"github.com/DataDog/datadog-agent/pkg/logs/internal/tag"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
)

// Tailer tails a file, decodes the messages it contains, and passes them to a
// supplied output channel for further processing.
//
// Operational Overview
//
// Tailers have three components, organized as a pipeline.  The first,
// readForever, polls the file, trying to read more data.  That data is passed
// to the second component, the decoder.  The decoder produces
// decoder.Messages, which are passed to the third component, forwardMessages.
// This component translates the decoder.Messages into message.Messages and
// sends them to the tailer's output channel.
type Tailer struct {
	// lastReadOffset is the last file offset that was read.  This value must
	// be accessed atomically after the tailer is started (via
	// getLastReadOffset, setLastReadOffset, and incrementLastReadOffset)
	lastReadOffset int64

	// decodedOffset is the offset in the file at which the latest decoded message
	// ends.  TODO(dustin): this field is accessed both atomically and non-atomically.
	decodedOffset int64

	// bytesRead is the number of bytes successfully read from the file by this
	// tailer.  This may be smaller than lastReadOffset if the tailer did not
	// begin at the start of the file.
	bytesRead int64

	// file contains the logs configuration for the file to parse (path, source, ...)
	// If you are looking for the os.file use to read on the FS, see osFile.
	file *File

	// fullpath is the absolute path to file.Path.
	fullpath string

	// osFile is the os.File object from which log data is read.  The read implementation
	// is platform-specific.
	osFile *os.File

	// tags are the tags to be attached to each log message, excluding tags provided
	// by the tag provider.
	tags []string

	// tagProvider provides additional tags to be attached to each log message.  It
	// is called once for each log message.
	tagProvider tag.Provider

	// outputChan is the channel to which fully-decoded messages are written.
	outputChan chan *message.Message

	// decoder handles decoding the raw bytes read from the file into log messages.
	decoder *decoder.Decoder

	// sleepDuration is the time between polls of the underlying file.
	sleepDuration time.Duration

	// closeTimeout is the duration the tailer will remain active after its file
	// has been rotated.  This allows the tailer to complete reading and processing
	// any remaining log lines in the file.
	closeTimeout time.Duration

	// isFinished is an atomic value, set to 1 when the tailer has closed its input
	// and flushed all messages.
	isFinished int32

	// didFileRotate is an atomic value, used to determine hasFileRotated.
	didFileRotate int32

	// stop is monitored by the readForever component, and causes it to stop reading
	// and close the channel to the decoder.
	stop chan struct{}

	// done is closed when the forwardMessages component has forwarded all messages.
	done chan struct{}

	// forwardContext is the context for attempts to send completed messages to
	// the tailer's output channel.  Once this context is finished, messages may
	// be discarded.
	forwardContext context.Context

	// stopForward is the cancellation function for forwardContext.  This will
	// force the forwardMessages goroutine to stop, even if it is currently
	// blocked sending to the tailer's outputChan.
	stopForward context.CancelFunc
}

// NewTailer returns an initialized Tailer, read to be started.
//
// The resulting Tailer will read from the given `file`, decode the content
// with the given `decoder`, and send the resulting log messages to outputChan.
// The Tailer takes ownership of the decoder and will start and stop it as
// necessary.
//
// The Tailer must poll for content in the file.  The `sleepDuration` parameter
// specifies how long the tailer should wait between polls.
func NewTailer(outputChan chan *message.Message, file *File, sleepDuration time.Duration, decoder *decoder.Decoder) *Tailer {

	var tagProvider tag.Provider
	if file.Source.Config.Identifier != "" {
		tagProvider = tag.NewProvider(containers.BuildTaggerEntityName(file.Source.Config.Identifier))
	} else {
		tagProvider = tag.NewLocalProvider([]string{})
	}

	forwardContext, stopForward := context.WithCancel(context.Background())
	closeTimeout := coreConfig.Datadog.GetDuration("logs_config.close_timeout") * time.Second

	return &Tailer{
		file:           file,
		outputChan:     outputChan,
		decoder:        decoder,
		tagProvider:    tagProvider,
		lastReadOffset: 0,
		sleepDuration:  sleepDuration,
		closeTimeout:   closeTimeout,
		stop:           make(chan struct{}, 1),
		done:           make(chan struct{}, 1),
		forwardContext: forwardContext,
		stopForward:    stopForward,
	}
}

// NewRotatedTailer creates a new tailer that replaces this one, writing
// messages to the same channel but using an updated file and decoder.
func (t *Tailer) NewRotatedTailer(file *File, decoder *decoder.Decoder) *Tailer {
	return NewTailer(t.outputChan, file, t.sleepDuration, decoder)
}

// Identifier returns a string that identifies this tailer in the registry.
func (t *Tailer) Identifier() string {
	// FIXME(remy): during container rotation, this Identifier() method could return
	// the same value for different tailers. It is happening during container rotation
	// where the dead container still has a tailer running on the log file, and the tailer
	// of the freshly spawned container starts tailing this file as well.
	//
	// This is the identifier used in the registry, so changing it will invalidate existing
	// registry entries on upgrade.
	return fmt.Sprintf("file:%s", t.file.Path)
}

// Start begins the tailer's operation in a dedicated goroutine.
func (t *Tailer) Start(offset int64, whence int) error {
	err := t.setup(offset, whence)
	if err != nil {
		t.file.Source.Status.Error(err)
		return err
	}
	t.file.Source.Status.Success()
	t.file.Source.AddInput(t.file.Path)

	go t.forwardMessages()
	t.decoder.Start()
	go t.readForever()

	return nil
}

// StartFromBeginning is a shortcut to start the tailer at the beginning of the
// file.
func (t *Tailer) StartFromBeginning() error {
	return t.Start(0, io.SeekStart)
}

// Stop stops the tailer and returns only after all in-flight messages have
// been flushed to the output channel.
func (t *Tailer) Stop() {
	t.stop <- struct{}{}
	t.file.Source.RemoveInput(t.file.Path)
	// wait for the decoder to be flushed
	<-t.done
}

// StopAfterFileRotation prepares the tailer to stop after a timeout
// to finish reading its file that has been log-rotated
func (t *Tailer) StopAfterFileRotation() {
	t.fileHasRotated()
	go func() {
		time.Sleep(t.closeTimeout)
		t.stopForward()
		t.stop <- struct{}{}
	}()
	t.file.Source.RemoveInput(t.file.Path)
}

// readForever lets the tailer tail the content of a file
// until it is closed or the tailer is stopped.
func (t *Tailer) readForever() {
	defer func() {
		t.osFile.Close()
		t.decoder.Stop()
		log.Info("Closed", t.file.Path, "for tailer key", t.file.GetScanKey(), "read", t.bytesRead, "bytes and", t.decoder.GetLineCount(), "lines")
	}()

	for {
		n, err := t.read()
		if err != nil {
			return
		}
		t.recordBytes(int64(n))

		select {
		case <-t.stop:
			if n != 0 && t.hasFileRotated() {
				log.Warn("Tailer stopped after rotation close timeout with remaining unread data")
			}
			// stop reading data from file
			return
		default:
			if n == 0 {
				// wait for new data to come
				t.wait()
			}
		}
	}
}

// buildTailerTags groups the file tag, directory (if wildcard path) and user tags
func (t *Tailer) buildTailerTags() []string {
	tags := []string{fmt.Sprintf("filename:%s", filepath.Base(t.file.Path))}
	if t.file.IsWildcardPath {
		tags = append(tags, fmt.Sprintf("dirname:%s", filepath.Dir(t.file.Path)))
	}
	return tags
}

// IsFinished returns true if the tailer has flushed all messages to the output
// channel, either because it has been stopped or because of an error reading from
// the input file.
func (t *Tailer) IsFinished() bool {
	return atomic.LoadInt32(&t.isFinished) != 0
}

// forwardMessages lets the Tailer forward log messages to the output channel
func (t *Tailer) forwardMessages() {
	defer func() {
		// the decoder has successfully been flushed
		atomic.StoreInt32(&t.isFinished, 1)
		close(t.done)
	}()
	for output := range t.decoder.OutputChan {
		offset := t.decodedOffset + int64(output.RawDataLen)
		identifier := t.Identifier()
		if t.hasFileRotated() {
			offset = 0
			identifier = ""
		}
		t.decodedOffset = offset
		origin := message.NewOrigin(t.file.Source)
		origin.Identifier = identifier
		origin.Offset = strconv.FormatInt(offset, 10)
		origin.SetTags(append(t.tags, t.tagProvider.GetTags()...))
		// Ignore empty lines once the registry offset is updated
		if len(output.Content) == 0 {
			continue
		}
		// Make the write to the output chan cancellable to be able to stop the tailer
		// after a file rotation when it is stuck on it.
		// We don't return directly to keep the same shutdown sequence that in the
		// normal case.
		select {
		case t.outputChan <- message.NewMessage(output.Content, origin, output.Status, output.IngestionTimestamp):
		case <-t.forwardContext.Done():
		}
	}
}

// incrementLastReadOffset increments the lastReadOffset field, atomically.
func (t *Tailer) incrementLastReadOffset(n int) {
	atomic.AddInt64(&t.lastReadOffset, int64(n))
}

// getLastReadOffset gets the value of lastReadOffset, atomically.
func (t *Tailer) getLastReadOffset() int64 {
	return atomic.LoadInt64(&t.lastReadOffset)
}

// GetDetectedPattern returns the decoder's detected pattern.
func (t *Tailer) GetDetectedPattern() *regexp.Regexp {
	return t.decoder.GetDetectedPattern()
}

// fileHasRotated causes subsequent calls to hasFileRotated to return true.
func (t *Tailer) fileHasRotated() {
	atomic.StoreInt32(&t.didFileRotate, 1)
}

// hasFileRotated returns true if the file has been rotated, and this tailer replaced
// with a new tailer for the new file.
func (t *Tailer) hasFileRotated() bool {
	return atomic.LoadInt32(&t.didFileRotate) != 0
}

// wait lets the tailer sleep for a bit
func (t *Tailer) wait() {
	time.Sleep(t.sleepDuration)
}

func (t *Tailer) recordBytes(n int64) {
	t.bytesRead += n
	t.file.Source.BytesRead.Add(n)
	if t.file.Source.ParentSource != nil {
		t.file.Source.ParentSource.BytesRead.Add(n)
	}
}