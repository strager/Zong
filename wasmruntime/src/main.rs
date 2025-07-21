use std::env;
use std::fs;
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

    // Create imports array with print function in "env" module
    let imports = [print_func.into()];

    // Instantiate the module
    let instance = Instance::new(&mut store, &module, &imports)?;

    // Get the main function export and call it
    let main_func = instance.get_typed_func::<(), ()>(&mut store, "main")?;
    main_func.call(&mut store, ())?;

    Ok(())
}