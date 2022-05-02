// Code generated - DO NOT EDIT.
// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux
// +build linux

package probe

// Syscall represents a syscall identifier
type Syscall int

// Linux syscall identifiers
const (
	SysRead                  Syscall = 3
	SysWrite                 Syscall = 4
	SysOpen                  Syscall = 5
	SysClose                 Syscall = 6
	SysWaitpid               Syscall = 7
	SysCreat                 Syscall = 8
	SysLink                  Syscall = 9
	SysUnlink                Syscall = 10
	SysChdir                 Syscall = 12
	SysMknod                 Syscall = 14
	SysChmod                 Syscall = 15
	SysLchown                Syscall = 16
	SysBreak                 Syscall = 17
	SysLseek                 Syscall = 19
	SysGetpid                Syscall = 20
	SysSetuid                Syscall = 23
	SysGetuid                Syscall = 24
	SysAlarm                 Syscall = 27
	SysStty                  Syscall = 31
	SysGtty                  Syscall = 32
	SysAccess                Syscall = 33
	SysNice                  Syscall = 34
	SysFtime                 Syscall = 35
	SysSync                  Syscall = 36
	SysKill                  Syscall = 37
	SysRename                Syscall = 38
	SysMkdir                 Syscall = 39
	SysRmdir                 Syscall = 40
	SysDup                   Syscall = 41
	SysPipe                  Syscall = 42
	SysTimes                 Syscall = 43
	SysProf                  Syscall = 44
	SysBrk                   Syscall = 45
	SysSetgid                Syscall = 46
	SysGetgid                Syscall = 47
	SysGeteuid               Syscall = 49
	SysGetegid               Syscall = 50
	SysLock                  Syscall = 53
	SysIoctl                 Syscall = 54
	SysFcntl                 Syscall = 55
	SysMpx                   Syscall = 56
	SysSetpgid               Syscall = 57
	SysUlimit                Syscall = 58
	SysUmask                 Syscall = 60
	SysChroot                Syscall = 61
	SysDup2                  Syscall = 63
	SysGetppid               Syscall = 64
	SysGetpgrp               Syscall = 65
	SysSetsid                Syscall = 66
	SysSgetmask              Syscall = 68
	SysSsetmask              Syscall = 69
	SysSetreuid              Syscall = 70
	SysSetregid              Syscall = 71
	SysSethostname           Syscall = 74
	SysSetrlimit             Syscall = 75
	SysGetrusage             Syscall = 77
	SysGettimeofday          Syscall = 78
	SysSettimeofday          Syscall = 79
	SysGetgroups             Syscall = 80
	SysSetgroups             Syscall = 81
	SysSymlink               Syscall = 83
	SysReadlink              Syscall = 85
	SysMmap                  Syscall = 90
	SysMunmap                Syscall = 91
	SysTruncate              Syscall = 92
	SysFtruncate             Syscall = 93
	SysFchmod                Syscall = 94
	SysFchown                Syscall = 95
	SysGetpriority           Syscall = 96
	SysSetpriority           Syscall = 97
	SysProfil                Syscall = 98
	SysIoperm                Syscall = 101
	SysSocketcall            Syscall = 102
	SysSyslog                Syscall = 103
	SysSetitimer             Syscall = 104
	SysGetitimer             Syscall = 105
	SysStat                  Syscall = 106
	SysLstat                 Syscall = 107
	SysFstat                 Syscall = 108
	SysIopl                  Syscall = 110
	SysVhangup               Syscall = 111
	SysIdle                  Syscall = 112
	SysVm86                  Syscall = 113
	SysWait4                 Syscall = 114
	SysSysinfo               Syscall = 116
	SysFsync                 Syscall = 118
	SysSetdomainname         Syscall = 121
	SysUname                 Syscall = 122
	SysModifyLdt             Syscall = 123
	SysMprotect              Syscall = 125
	SysCreateModule          Syscall = 127
	SysGetKernelSyms         Syscall = 130
	SysGetpgid               Syscall = 132
	SysFchdir                Syscall = 133
	SysBdflush               Syscall = 134
	SysSysfs                 Syscall = 135
	SysAfsSyscall            Syscall = 137
	SysSetfsuid              Syscall = 138
	SysSetfsgid              Syscall = 139
	SysLlseek                Syscall = 140
	SysGetdents              Syscall = 141
	SysNewselect             Syscall = 142
	SysFlock                 Syscall = 143
	SysMsync                 Syscall = 144
	SysReadv                 Syscall = 145
	SysWritev                Syscall = 146
	SysGetsid                Syscall = 147
	SysFdatasync             Syscall = 148
	SysMlock                 Syscall = 150
	SysMunlock               Syscall = 151
	SysMlockall              Syscall = 152
	SysMunlockall            Syscall = 153
	SysSchedSetparam         Syscall = 154
	SysSchedGetparam         Syscall = 155
	SysSchedSetscheduler     Syscall = 156
	SysSchedGetscheduler     Syscall = 157
	SysSchedYield            Syscall = 158
	SysSchedGetPriorityMax   Syscall = 159
	SysSchedGetPriorityMin   Syscall = 160
	SysMremap                Syscall = 163
	SysSetresuid             Syscall = 164
	SysGetresuid             Syscall = 165
	SysQueryModule           Syscall = 166
	SysPoll                  Syscall = 167
	SysNfsservctl            Syscall = 168
	SysSetresgid             Syscall = 169
	SysGetresgid             Syscall = 170
	SysPrctl                 Syscall = 171
	SysPread64               Syscall = 179
	SysPwrite64              Syscall = 180
	SysChown                 Syscall = 181
	SysGetcwd                Syscall = 182
	SysCapget                Syscall = 183
	SysCapset                Syscall = 184
	SysGetpmsg               Syscall = 187
	SysPutpmsg               Syscall = 188
	SysUgetrlimit            Syscall = 190
	SysReadahead             Syscall = 191
	SysMultiplexer           Syscall = 201
	SysGetdents64            Syscall = 202
	SysPivotRoot             Syscall = 203
	SysMadvise               Syscall = 205
	SysMincore               Syscall = 206
	SysGettid                Syscall = 207
	SysTkill                 Syscall = 208
	SysSetxattr              Syscall = 209
	SysLsetxattr             Syscall = 210
	SysFsetxattr             Syscall = 211
	SysGetxattr              Syscall = 212
	SysLgetxattr             Syscall = 213
	SysFgetxattr             Syscall = 214
	SysListxattr             Syscall = 215
	SysLlistxattr            Syscall = 216
	SysFlistxattr            Syscall = 217
	SysRemovexattr           Syscall = 218
	SysLremovexattr          Syscall = 219
	SysFremovexattr          Syscall = 220
	SysSchedSetaffinity      Syscall = 222
	SysSchedGetaffinity      Syscall = 223
	SysTuxcall               Syscall = 225
	SysIoSetup               Syscall = 227
	SysIoDestroy             Syscall = 228
	SysIoSubmit              Syscall = 230
	SysIoCancel              Syscall = 231
	SysFadvise64             Syscall = 233
	SysEpollCreate           Syscall = 236
	SysEpollCtl              Syscall = 237
	SysEpollWait             Syscall = 238
	SysRemapFilePages        Syscall = 239
	SysTimerCreate           Syscall = 240
	SysTimerGetoverrun       Syscall = 243
	SysTimerDelete           Syscall = 244
	SysTgkill                Syscall = 250
	SysStatfs64              Syscall = 252
	SysFstatfs64             Syscall = 253
	SysRtas                  Syscall = 255
	SysUnshare               Syscall = 282
	SysSplice                Syscall = 283
	SysTee                   Syscall = 284
	SysVmsplice              Syscall = 285
	SysOpenat                Syscall = 286
	SysMkdirat               Syscall = 287
	SysMknodat               Syscall = 288
	SysFchownat              Syscall = 289
	SysUnlinkat              Syscall = 292
	SysRenameat              Syscall = 293
	SysLinkat                Syscall = 294
	SysSymlinkat             Syscall = 295
	SysReadlinkat            Syscall = 296
	SysFchmodat              Syscall = 297
	SysFaccessat             Syscall = 298
	SysGetRobustList         Syscall = 299
	SysSetRobustList         Syscall = 300
	SysMovePages             Syscall = 301
	SysGetcpu                Syscall = 302
	SysSignalfd              Syscall = 305
	SysTimerfdCreate         Syscall = 306
	SysEventfd               Syscall = 307
	SysSyncFileRange2        Syscall = 308
	SysSignalfd4             Syscall = 313
	SysEventfd2              Syscall = 314
	SysEpollCreate1          Syscall = 315
	SysDup3                  Syscall = 316
	SysPipe2                 Syscall = 317
	SysPerfEventOpen         Syscall = 319
	SysPreadv                Syscall = 320
	SysPwritev               Syscall = 321
	SysPrlimit64             Syscall = 325
	SysSocket                Syscall = 326
	SysBind                  Syscall = 327
	SysConnect               Syscall = 328
	SysListen                Syscall = 329
	SysAccept                Syscall = 330
	SysGetsockname           Syscall = 331
	SysGetpeername           Syscall = 332
	SysSocketpair            Syscall = 333
	SysSend                  Syscall = 334
	SysSendto                Syscall = 335
	SysRecv                  Syscall = 336
	SysRecvfrom              Syscall = 337
	SysShutdown              Syscall = 338
	SysSetsockopt            Syscall = 339
	SysGetsockopt            Syscall = 340
	SysSendmsg               Syscall = 341
	SysRecvmsg               Syscall = 342
	SysAccept4               Syscall = 344
	SysNameToHandleAt        Syscall = 345
	SysOpenByHandleAt        Syscall = 346
	SysSyncfs                Syscall = 348
	SysSendmmsg              Syscall = 349
	SysSetns                 Syscall = 350
	SysSchedSetattr          Syscall = 355
	SysSchedGetattr          Syscall = 356
	SysRenameat2             Syscall = 357
	SysSeccomp               Syscall = 358
	SysGetrandom             Syscall = 359
	SysMemfdCreate           Syscall = 360
	SysBpf                   Syscall = 361
	SysUserfaultfd           Syscall = 364
	SysMembarrier            Syscall = 365
	SysPreadv2               Syscall = 380
	SysPwritev2              Syscall = 381
	SysSemget                Syscall = 393
	SysSemctl                Syscall = 394
	SysShmget                Syscall = 395
	SysShmctl                Syscall = 396
	SysShmat                 Syscall = 397
	SysShmdt                 Syscall = 398
	SysMsgget                Syscall = 399
	SysMsgsnd                Syscall = 400
	SysMsgrcv                Syscall = 401
	SysMsgctl                Syscall = 402
	SysPidfdSendSignal       Syscall = 424
	SysIoUringSetup          Syscall = 425
	SysIoUringEnter          Syscall = 426
	SysIoUringRegister       Syscall = 427
	SysOpenTree              Syscall = 428
	SysMoveMount             Syscall = 429
	SysFsopen                Syscall = 430
	SysFsconfig              Syscall = 431
	SysFsmount               Syscall = 432
	SysFspick                Syscall = 433
	SysPidfdOpen             Syscall = 434
	SysCloseRange            Syscall = 436
	SysOpenat2               Syscall = 437
	SysPidfdGetfd            Syscall = 438
	SysFaccessat2            Syscall = 439
	SysProcessMadvise        Syscall = 440
	SysEpollPwait2           Syscall = 441
	SysMountSetattr          Syscall = 442
	SysQuotactlFd            Syscall = 443
	SysLandlockCreateRuleset Syscall = 444
	SysLandlockAddRule       Syscall = 445
	SysLandlockRestrictSelf  Syscall = 446
)