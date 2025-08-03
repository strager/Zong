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

    // Create the read_line function that will be imported by the WASM module
    let read_line_func = Func::new(
        &mut store,
        FuncType::new(&engine, [ValType::I32], []),
        |mut caller, params, _results| {
            use std::io::{self, BufRead};
            
            // Get destination address from parameter
            let dest_addr = params[0].unwrap_i32() as usize;
            
            // Read a line from stdin
            let stdin = io::stdin();
            let mut line = String::new();
            match stdin.lock().read_line(&mut line) {
                Ok(_) => {
                    // Convert to bytes
                    let input_bytes = line.as_bytes();
                    let input_len = input_bytes.len() as u64;
                    
                    // Get memory and tstack global
                    let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
                    let tstack_global = caller.get_export("tstack").unwrap().into_global().unwrap();
                    let current_tstack = tstack_global.get(&mut caller).unwrap_i32() as usize;
                    
                    // Allocate space for input bytes on tstack
                    let input_ptr = current_tstack as u32;
                    
                    // Write input bytes to tstack
                    memory.data_mut(&mut caller)[current_tstack..current_tstack + input_bytes.len()]
                        .copy_from_slice(input_bytes);
                    
                    // Update tstack global to point past the input bytes
                    let new_tstack = (current_tstack + input_bytes.len()) as i32;
                    tstack_global.set(&mut caller, new_tstack.into()).unwrap();
                    
                    // Write slice structure to the destination address: [items_ptr: i32 at offset 0, length: i64 at offset 8]
                    let data = memory.data_mut(&mut caller);
                    
                    // items_ptr (i32) at offset 0
                    data[dest_addr..dest_addr + 4].copy_from_slice(&input_ptr.to_le_bytes());
                    
                    // length (i64) at offset 8
                    data[dest_addr + 8..dest_addr + 16].copy_from_slice(&input_len.to_le_bytes());
                    
                    Ok(())
                },
                Err(_) => {
                    // On error, write empty slice to destination
                    let memory = caller.get_export("memory").unwrap().into_memory().unwrap();
                    let data = memory.data_mut(&mut caller);
                    
                    // items_ptr = 0 (null pointer)
                    data[dest_addr..dest_addr + 4].copy_from_slice(&0u32.to_le_bytes());
                    
                    // length = 0
                    data[dest_addr + 8..dest_addr + 16].copy_from_slice(&0u64.to_le_bytes());
                    
                    Ok(())
                }
            }
        },
    );

    // Create imports array - order must match WASM import order: print, print_bytes, read_line functions
    // tstack global is now defined in the WASM module itself, not imported
    let imports = [print_func.into(), print_bytes_func.into(), read_line_func.into()];

    // Instantiate the module
    let instance = Instance::new(&mut store, &module, &imports)?;

    // Get the main function export and call it
    let main_func = instance.get_typed_func::<(), ()>(&mut store, "main")?;
    main_func.call(&mut store, ())?;

    Ok(())
}