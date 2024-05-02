# slbool

`slbool` defines a HLSL and Go friendly `int32` Bool type.  The standard HLSL bool type causes obscure errors, and the int32 obeys the 4 byte basic alignment requirements.

`gosl` automatically converts this Go code into appropriate HLSL code.


