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

#ifdef __cplusplus
extern "C" {
#endif

// Base Arena Structure
typedef struct {
    uint8_t* ptr;
    size_t   len;
    size_t   offset;
} wuffs_base__arena;

static inline wuffs_base__arena wuffs_base__make_arena(uint8_t* ptr, size_t len) {
    wuffs_base__arena ret;
    ret.ptr = ptr;
    ret.len = len;
    ret.offset = 0;
    return ret;
}

static inline uint64_t wuffs_base__arena__mark(wuffs_base__arena* self) {
    return (uint64_t)self->offset;
}

static inline void wuffs_base__arena__release(wuffs_base__arena* self, uint64_t mark) {
    if (mark <= self->offset) {
        self->offset = (size_t)mark;
    }
}

// Base Vector Structure
typedef struct {
    uint8_t* data;
    uint32_t len;
    uint32_t cap;
    uint32_t elem_size;
} wuffs_base__vec;

static inline uint32_t wuffs_base__vec__length(wuffs_base__vec* self) {
    return self ? self->len : 0;
}

static inline uint32_t wuffs_base__vec__capacity(wuffs_base__vec* self) {
    return self ? self->cap : 0;
}

// Base String View Structure
typedef struct {
    const uint8_t* ptr;
    uint32_t       len;
} wuffs_base__str;

static inline uint32_t wuffs_base__str__length(wuffs_base__str* self) {
    return self ? self->len : 0;
}

// Environment Capability Structure
typedef struct {
    int argc;
    char** argv;
} wuffs_os__environment;

static inline wuffs_os__environment wuffs_os__make_environment(int argc, char** argv) {
    wuffs_os__environment env;
    env.argc = argc;
    env.argv = argv;
    return env;
}

#ifdef __cplusplus
}
#endif

#endif  // WUFFS_OS_H
