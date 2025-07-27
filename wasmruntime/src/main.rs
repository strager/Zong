use std::env;
use std::fs;
use std::io::{self, Write};
use wasmtime::*;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let args: Vec<String> = env::args().collect();
    if args.len() != 2 {
        eprintln!("Usage: {} <wasm-file>", args[0]);
        std::process::exit(1);
    }

    let wasm_file = &args[1];
    let wasm_bytes = fs::read(wasm_file)?;

    // Create Wasmtime engine, store, and module
    let engine = Engine::default();
    let mut store = Store::new(&engine, ());
    let module = Module::new(&engine, &wasm_bytes)?;

    // Create the print function that will be imported by the WASM module
    let print_func = Func::wrap(&mut store, |n: i64| {
        println!("{}", n);
    });

    // Create the print_bytes function that will be imported by the WASM module
    let print_bytes_func = Func::new(
        &mut store,
        FuncType::new(&engine, [ValType::I32], []),
        |mut caller, params, _results| {
            let slice_ptr = params[0].unwrap_i32();
            
            // Read slice structure from WASM memory
            let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
            let data = memory.data(&caller);
            
            // Slice structure: [items_ptr: i32, length: i64]
            let items_ptr = u32::from_le_bytes([
                data[slice_ptr as usize],
                data[slice_ptr as usize + 1],
                data[slice_ptr as usize + 2],
                data[slice_ptr as usize + 3],
            ]);
            
            let length = u64::from_le_bytes([
                data[slice_ptr as usize + 8],
                data[slice_ptr as usize + 9],
                data[slice_ptr as usize + 10],
                data[slice_ptr as usize + 11],
                data[slice_ptr as usize + 12],
                data[slice_ptr as usize + 13],
                data[slice_ptr as usize + 14],
                data[slice_ptr as usize + 15],
            ]);
            
            // Read string bytes from memory
            let string_bytes = &data[items_ptr as usize..(items_ptr as usize + length as usize)];
            
            // Write raw bytes to stdout (no trailing newline)
            io::stdout().write_all(string_bytes).unwrap();
            
            Ok(())
        },
    );

    // Create imports array - order must match WASM import order: print, print_bytes functions
    // tstack global is now defined in the WASM module itself, not imported
    let imports = [print_func.into(), print_bytes_func.into()];

    // Instantiate the module
    let instance = Instance::new(&mut store, &module, &imports)?;

    // Get the main function export and call it
    let main_func = instance.get_typed_func::<(), ()>(&mut store, "main")?;
    main_func.call(&mut store, ())?;

    Ok(())
}