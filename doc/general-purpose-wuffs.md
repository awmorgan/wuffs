# General-Purpose Wuffs

Wuffs can be used to write safe, high-performance file format decoders/encoders as well as standalone general-purpose command-line and system applications.

General-purpose Wuffs strictly preserves Wuffs' foundational guarantees:
- **Provable Memory Safety**: Mathematical interval arithmetic and bounds checking.
- **Zero Runtime Garbage Collection**: Zero dynamic heap allocations or hidden pointer chasing.
- **Bounded Memory Execution**: Applications run within caller-provided memory regions or stack-allocated arenas.

---

## Key Language Constructs

### 1. Capability Security Model (`base.env`)
Standalone Wuffs applications define a `main!` entry point that receives system capabilities via `base.env`:

```wuffs
pub func main!(env: base.env) base.status {
    var stdout : base.io_writer
    stdout = args.env.stdout()
    // Perform safe streaming I/O...
    return ok
}
```

### 2. Stack-Disciplined Memory Arenas (`base.arena`)
To manage temporary memory without dynamic allocation or raw pointers, Wuffs uses **Bounded Memory Arenas** with LIFO mark/release checkpoints:

```wuffs
pub func main!(env: base.env) base.status {
    var storage : base.arena[4096]
    
    // Save checkpoint
    var mark : base.u64
    mark = storage.mark()
    
    // Perform operations...
    
    // Restore checkpoint
    storage.release!(mark: mark)
    return ok
}
```

### 3. Handle-Based Vectors (`base.vec`)
Dynamic collections are represented as bounded vectors (`base.vec`). Vector capacity is constrained at compile-time to maintain range-proof facts:

```wuffs
var v : base.vec[base.u32, 16]
```

### 4. UTF-8 String Views (`base.str`)
`base.str` provides UTF-8 validated views over byte slices:

```wuffs
var msg : base.str = "Hello Wuffs"
```

---

## Compiling & Testing

### C Transpilation
To transpile a standalone Wuffs program to C:
```bash
wuffs-c gen -package_name=main main.wuffs > main.c
```

### C Compiler Requirements
Compiling transpiled C binaries requires a C99-compliant compiler:
- **Clang** (`clang` v10+)
- **GCC** (`gcc` v7+)
- **MSVC** (`cl` from Visual Studio Command Prompt)

The compiler can be overridden using the `CC` environment variable:
```bash
export CC=clang
go test ./test -v
```
