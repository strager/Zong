# Extern Function Tests

## Test: basic extern function declaration
```zong-program
extern "wasi_snapshot_preview1" {
    func fd_write(fd: I32, iovs: I32, iovs_len: I32, nwritten: I32): I32;
}

func main() {
    print(42);
}
```
```ast
[
    (extern "wasi_snapshot_preview1" (
        (func "fd_write" [(param "fd" "I32" named) (param "iovs" "I32" named) (param "iovs_len" "I32" named) (param "nwritten" "I32" named)] "I32" nil)
    ))
    (func "main" [] nil [
        (call (var "print") 42)
    ])
]
```

## Test: multiple extern functions in one block
```zong-program
extern "wasi_snapshot_preview1" {
    func fd_write(fd: I32, iovs: I32, iovs_len: I32, nwritten: I32): I32;
    func random_get(buf: I32, buf_len: I32): I32;
}

func main() {
    print(123);
}
```
```ast
[
    (extern "wasi_snapshot_preview1" (
        (func "fd_write" [(param "fd" "I32" named) (param "iovs" "I32" named) (param "iovs_len" "I32" named) (param "nwritten" "I32" named)] "I32" nil)
        (func "random_get" [(param "buf" "I32" named) (param "buf_len" "I32" named)] "I32" nil)
    ))
    (func "main" [] nil [
        (call (var "print") 123)
    ])
]
```

## Test: extern function with no parameters
```zong-program
extern "wasi_snapshot_preview1" {
    func clock_time_get(): I64;
}

func main() {
    print(456);
}
```
```ast
[
    (extern "wasi_snapshot_preview1" (
        (func "clock_time_get" [] "I64" nil)
    ))
    (func "main" [] nil [
        (call (var "print") 456)
    ])
]
```

## Test: extern function with void return type
```zong-program
extern "wasi_snapshot_preview1" {
    func proc_exit(code: I32);
}

func main() {
    print(789);
}
```
```ast
[
    (extern "wasi_snapshot_preview1" (
        (func "proc_exit" [(param "code" "I32" named)] nil nil)
    ))
    (func "main" [] nil [
        (call (var "print") 789)
    ])
]
```

## Test: multiple extern blocks with different modules
```zong-program
extern "wasi_snapshot_preview1" {
    func fd_write(fd: I32, iovs: I32, iovs_len: I32, nwritten: I32): I32;
}

extern "env" {
    func custom_func(x: I64): I64;
}

func main() {
    print(999);
}
```
```ast
[
    (extern "wasi_snapshot_preview1" (
        (func "fd_write" [(param "fd" "I32" named) (param "iovs" "I32" named) (param "iovs_len" "I32" named) (param "nwritten" "I32" named)] "I32" nil)
    ))
    (extern "env" (
        (func "custom_func" [(param "x" "I64" named)] "I64" nil)
    ))
    (func "main" [] nil [
        (call (var "print") 999)
    ])
]
```

## Test: calling WASI function with return value
```zong-program
extern "wasi_snapshot_preview1" {
    func clock_time_get(id: I32, precision: I64, time_ptr: I32): I32;
}

func main() {
    var result: I32 = clock_time_get(0, 1, 0);
    print(result);
}
```
```execute
0
```

## Test: using random_get WASI function
```zong-program
extern "wasi_snapshot_preview1" {
    func random_get(buf: I32, buf_len: I32): I32;
}

func main() {
    var result: I32 = random_get(0, 0);
    print(result);
}
```
```execute
0
```

## Test: extern syntax error - missing module name
```zong-program
extern {
    func bad_func(): I32;
}
```
```compile-error
expected string literal for extern module name
```

## Test: extern syntax error - missing function body braces
```zong-program
extern "test_module"
    func bad_func(): I32;
```
```compile-error
expected '{' after extern module name
expected '{' for function body
unexpected token ';' in expression
```

## Test: extern syntax error - missing semicolon after function declaration
```zong-program
extern "test_module" {
    func bad_func(): I32
}
```
```compile-error
expected ';' after extern function declaration
```

## Test: extern function with complex parameter types
```zong-program
extern "wasi_snapshot_preview1" {
    func clock_time_get(id: I32, precision: I64, time_ptr: I32): I32;
}

func main() {
    print(123);
}
```
```ast
[
    (extern "wasi_snapshot_preview1" (
        (func "clock_time_get" [(param "id" "I32" named) (param "precision" "I64" named) (param "time_ptr" "I32" named)] "I32" nil)
    ))
    (func "main" [] nil [
        (call (var "print") 123)
    ])
]
```
