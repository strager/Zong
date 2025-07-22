# Memory model

This document explains how memory is implemented in Zong.

## tstack pointer

The **tstack** is a linear region of memory. It has unbounded size.

The term "tstack" is short for "thread stack".

The **tstack pointer** is a pointer to the top of the tstack. The tstack pointer
is initialized to the base of the tstack. The base of the tstack is address 0.

## dstack pointer

The **dstack** is a linear region of memory. It has unbounded size.

The term "dstack" is short for "data stack".

The **dstack pointer** is a pointer to the top of the dstack. The dstack pointer
is initialized to the base of the dstack.

The dstack is not implemented. It is explained here for added confusion. Sorry.

## frame pointer

The **frame pointer** points within the tstack.

The frame pointer is computed on entry of each function and is stable within a
function call.

## address-of operator

The `&` unary operator is the address-of operator. Its operation depends on what
we are taking the address of.

### lvalue

If we are taking the address of an lvalue (variable):

During locals allocation (done once per `&`d variable, not once per `&` operation):

1. Allocate space in the function's frame.
2. Set the variable's frame pointer offset to a unique value.

On function entry (done once per function, not once per `&` operation):

1. Initialize the frame pointer local to the tstack pointer.
2. Increment the tstack pointer by the size of all `&`d locals

On `&` (each operation), load the frame pointer (WebAssembly local) and add the
local's frame pointer offset (constant).

### rvalue

If we are taking the address of an rvalue (computed or literal):

1. Store the value at the tstack pointer.
2. Increment the tstack pointer by the size of the stored data.
   - NOTE: Because Zong only supports I64 and I64* right now, and because both have size 8, the tstack pointer is always incremented by 8.
