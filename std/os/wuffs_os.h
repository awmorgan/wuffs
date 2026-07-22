// Copyright 2026 The Wuffs Authors.
//
// Licensed under the Apache License, Version 2.0 <LICENSE-APACHE or
// https://www.apache.org/licenses/LICENSE-2.0> or the MIT license
// <LICENSE-MIT or https://opensource.org/licenses/MIT>, at your
// option. This file may not be copied, modified, or distributed
// except according to those terms.
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

#ifndef WUFFS_OS_H
#define WUFFS_OS_H

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>
#include <string.h>

#ifdef _WIN32
#include <io.h>
#include <fcntl.h>
#endif

#ifdef __cplusplus
extern "C" {
#endif

static inline uint64_t wuffs_base__arena__mark(wuffs_base__arena* self) {
    return self ? (uint64_t)self->offset : 0;
}

static inline void wuffs_base__arena__release(wuffs_base__arena* self, uint64_t mark) {
    if (self && mark <= self->offset) {
        self->offset = (size_t)mark;
    }
}

static inline uint32_t wuffs_base__vec__length(wuffs_base__vec* self) {
    return self ? self->len : 0;
}

static inline uint32_t wuffs_base__vec__capacity(wuffs_base__vec* self) {
    return self ? self->cap : 0;
}

static inline uint32_t wuffs_base__str__length(wuffs_base__str* self) {
    return self ? self->len : 0;
}

static inline wuffs_base__str wuffs_base__str__from_c_string(const uint8_t* p) {
    wuffs_base__str ret;
    ret.ptr = p;
    ret.len = p ? (uint32_t)strlen((const char*)p) : 0;
    return ret;
}

static inline wuffs_base__slice_u8 wuffs_base__slice_u8__from_ptr_len(uint8_t* p, uint32_t len) {
    wuffs_base__slice_u8 ret;
    ret.ptr = p;
    ret.len = len;
    return ret;
}

static inline wuffs_base__env wuffs_os__make_environment(int argc, char** argv) {
#ifdef _WIN32
    _setmode(_fileno(stdout), _O_BINARY);
    _setmode(_fileno(stderr), _O_BINARY);
#endif
    wuffs_base__env env;
    env.argc = argc;
    env.argv = argv;
    return env;
}

static inline wuffs_base__io_buffer* wuffs_base__env__stdout(wuffs_base__env* self) {
    (void)self;
    static wuffs_base__io_buffer buf;
    return &buf;
}

#ifdef __cplusplus
}
#endif

#endif  // WUFFS_OS_H
