// /Users/oreilly/goki/pi/langs/golang/go.pig Lexer

// InCommentMulti all CurState must be at top!  any multi-line requires state 
InCommentMulti:		 CommentMultiline		 if CurState == "CommentMulti" {
    EndMulti:                CommentMultiline       if String == "*/"   do: PopState; Next; 
    StartEmbededMulti:       CommentMultiline       if String == "/*"   do: PushState: CommentMulti; Next; 
    Comment:                 CommentMultiline       if AnyRune          do: Next; 
}
// InStrBacktick curstate at start -- multiline requires state 
InStrBacktick:		 LitStrBacktick		 if CurState == "StrBacktick" {
    QuotedStrBacktick:       LitStrBacktick       if String == "\`"   do: Next; 
    EndStrBacktick:          LitStrBacktick       if String == "`"    do: PopState; Next; 
    StrBacktick:             LitStrBacktick       if AnyRune          do: Next; 
}
StartCommentMulti:		 CommentMultiline		 if String == "/*"	 do: PushState: CommentMulti; Next; 
LitStrBacktick:		 LitStrBacktick		 if String == "`"	 do: PushState: StrBacktick; Next; 
CommentLine:		 Comment		 if String == "//"	 do: EOL; 
SkipWhite:		 TextWhitespace		 if WhiteSpace	 do: Next; 
Letter:		 None		 if Letter {
    // Keyword this group should contain all reserved keywords 
    Keyword:       None       if Letter {
        break:             Keyword       if StrName == "break"         do: Name; 
        case:              Keyword       if StrName == "case"          do: Name; 
        chan:              Keyword       if StrName == "chan"          do: Name; 
        const:             Keyword       if StrName == "const"         do: Name; 
        continue:          Keyword       if StrName == "continue"      do: Name; 
        default:           Keyword       if StrName == "default"       do: Name; 
        defer:             Keyword       if StrName == "defer"         do: Name; 
        else:              Keyword       if StrName == "else"          do: Name; 
        fallthrough:       Keyword       if StrName == "fallthrough"   do: Name; 
        for:               Keyword       if StrName == "for"           do: Name; 
        func:              Keyword       if StrName == "func"          do: Name; 
        go:                Keyword       if StrName == "go"            do: Name; 
        goto:              Keyword       if StrName == "goto"          do: Name; 
        if:                Keyword       if StrName == "if"            do: Name; 
        import:            Keyword       if StrName == "import"        do: Name; 
        interface:         Keyword       if StrName == "interface"     do: Name; 
        map:               Keyword       if StrName == "map"           do: Name; 
        make:              Keyword       if StrName == "make"          do: Name; 
        new:               Keyword       if StrName == "new"           do: Name; 
        package:           Keyword       if StrName == "package"       do: Name; 
        range:             Keyword       if StrName == "range"         do: Name; 
        return:            Keyword       if StrName == "return"        do: Name; 
        select:            Keyword       if StrName == "select"        do: Name; 
        struct:            Keyword       if StrName == "struct"        do: Name; 
        switch:            Keyword       if StrName == "switch"        do: Name; 
        type:              Keyword       if StrName == "type"          do: Name; 
        var:               Keyword       if StrName == "var"           do: Name; 
    }
    // Type this group should contain all basic types, and no types that are not built into the language 
    Type:       None       if Letter {
        bool:             KeywordType       if StrName == "bool"         do: Name; 
        byte:             KeywordType       if StrName == "byte"         do: Name; 
        complex64:        KeywordType       if StrName == "complex64"    do: Name; 
        complex128:       KeywordType       if StrName == "complex128"   do: Name; 
        error:            KeywordType       if StrName == "error"        do: Name; 
        float32:          KeywordType       if StrName == "float32"      do: Name; 
        float64:          KeywordType       if StrName == "float64"      do: Name; 
        int:              KeywordType       if StrName == "int"          do: Name; 
        int8:             KeywordType       if StrName == "int8"         do: Name; 
        int16:            KeywordType       if StrName == "int16"        do: Name; 
        int32:            KeywordType       if StrName == "int32"        do: Name; 
        int64:            KeywordType       if StrName == "int64"        do: Name; 
        rune:             KeywordType       if StrName == "rune"         do: Name; 
        string:           KeywordType       if StrName == "string"       do: Name; 
        uint:             KeywordType       if StrName == "uint"         do: Name; 
        uint8:            KeywordType       if StrName == "uint8"        do: Name; 
        uint16:           KeywordType       if StrName == "uint16"       do: Name; 
        uint32:           KeywordType       if StrName == "uint32"       do: Name; 
        uint64:           KeywordType       if StrName == "uint64"       do: Name; 
        uintptr:          KeywordType       if StrName == "uintptr"      do: Name; 
    }
    Builtins:       None       if String == "" {
        append:        NameBuiltin       if StrName == "append"    do: Name; 
        cap:           NameBuiltin       if StrName == "cap"       do: Name; 
        close:         NameBuiltin       if StrName == "close"     do: Name; 
        complex:       NameBuiltin       if StrName == "complex"   do: Name; 
        copy:          NameBuiltin       if StrName == "copy"      do: Name; 
        delete:        NameBuiltin       if StrName == "delete"    do: Name; 
        imag:          NameBuiltin       if StrName == "imag"      do: Name; 
        len:           NameBuiltin       if StrName == "len"       do: Name; 
        panic:         NameBuiltin       if StrName == "panic"     do: Name; 
        print:         NameBuiltin       if StrName == "print"     do: Name; 
        println:       NameBuiltin       if StrName == "println"   do: Name; 
        real:          NameBuiltin       if StrName == "real"      do: Name; 
        recover:       NameBuiltin       if StrName == "recover"   do: Name; 
        true:          NameBuiltin       if StrName == "true"      do: Name; 
        false:         NameBuiltin       if StrName == "false"     do: Name; 
        iota:          NameBuiltin       if StrName == "iota"      do: Name; 
        nil:           NameBuiltin       if StrName == "nil"       do: Name; 
    }
    Name:       Name       if Letter   do: Name; 
}
Number:		 LitNum		 if Digit	 do: Number; 
Dot:		 None		 if String == "." {
    // NextNum lookahead for number 
    NextNum:       LitNum       if +1:Digit   do: Number; 
    // NextDot lookahead for another dot -- ellipses 
    NextDot:       None       if +1:String == "." {
        Ellipsis:       OpListEllipsis       if +2:String == "."   do: Next; 
    }
    // Period default is just a plain . 
    Period:       PunctSepPeriod       if String == "."   do: Next; 
}
LitStrSingle:		 LitStrSingle		 if String == "'"	 do: QuotedRaw; 
LitStrDouble:		 LitStrDouble		 if String == """	 do: QuotedRaw; 
LParen:		 PunctGpLParen		 if String == "("	 do: Next; 
RParen:		 PunctGpRParen		 if String == ")"	 do: Next; 
LBrack:		 PunctGpLBrack		 if String == "["	 do: Next; 
RBrack:		 PunctGpRBrack		 if String == "]"	 do: Next; 
LBrace:		 PunctGpLBrace		 if String == "{"	 do: Next; 
RBrace:		 PunctGpRBrace		 if String == "}"	 do: Next; 
Comma:		 PunctSepComma		 if String == ","	 do: Next; 
Semi:		 PunctSepSemicolon		 if String == ";"	 do: Next; 
Colon:		 None		 if String == ":" {
    Define:       OpAsgnDefine        if +1:String == "="   do: Next; 
    Colon:        PunctSepColon       if String == ":"      do: Next; 
}
Plus:		 None		 if String == "+" {
    AsgnAdd:       OpMathAsgnAdd       if +1:String == "="   do: Next; 
    AsgnInc:       OpAsgnInc           if +1:String == "+"   do: Next; 
    Add:           OpMathAdd           if String == "+"      do: Next; 
}
Minus:		 None		 if String == "-" {
    AsgnSub:       OpMathAsgnSub       if +1:String == "="   do: Next; 
    AsgnDec:       OpAsgnDec           if +1:String == "-"   do: Next; 
    Sub:           OpMathSub           if String == "-"      do: Next; 
}
Mult:		 None		 if String == "*" {
    AsgnMul:       OpMathAsgnMul       if +1:String == "="   do: Next; 
    Mult:          OpMathMul           if String == "*"      do: Next; 
}
// Div comments already matched above.. 
Div:		 None		 if String == "/" {
    AsgnDiv:       OpMathAsgnDiv       if +1:String == "="   do: Next; 
    Div:           OpMathDiv           if String == "/"      do: Next; 
}
Rem:		 None		 if String == "%" {
    AsgnRem:       OpMathAsgnRem       if +1:String == "="   do: Next; 
    Rem:           OpMathRem           if String == "%"      do: Next; 
}
Xor:		 None		 if String == "^" {
    AsgnXor:       OpBitAsgnXor       if +1:String == "="   do: Next; 
    Xor:           OpBitXor           if String == "^"      do: Next; 
}
Rangle:		 None		 if String == ">" {
    GtEq:             OpRelGtEq       if +1:String == "="   do: Next; 
    ShiftRight:       None            if +1:String == ">" {
        AsgnShiftRight:       OpBitAsgnShiftRight       if +2:String == "="   do: Next; 
        ShiftRight:           OpBitShiftRight           if +1:String == ">"   do: Next; 
    }
    Greater:       OpRelGreater       if String == ">"   do: Next; 
}
Langle:		 None		 if String == "<" {
    LtEq:            OpRelLtEq         if +1:String == "="   do: Next; 
    AsgnArrow:       OpAsgnArrow       if +1:String == "-"   do: Next; 
    ShiftLeft:       None              if +1:String == "<" {
        AsgnShiftLeft:       OpBitAsgnShiftLeft       if +2:String == "="   do: Next; 
        ShiftLeft:           OpBitShiftLeft           if +1:String == "<"   do: Next; 
    }
    Less:       OpRelLess       if String == "<"   do: Next; 
}
Equals:		 None		 if String == "=" {
    Equality:       OpRelEqual         if +1:String == "="   do: Next; 
    Asgn:           OpAsgnAssign       if String == "="      do: Next; 
}
Not:		 None		 if String == "!" {
    NotEqual:       OpRelNotEqual       if +1:String == "="   do: Next; 
    Not:            OpLogNot            if String == "!"      do: Next; 
}
And:		 None		 if String == "&" {
    AsgnAnd:       OpBitAsgnAnd       if +1:String == "="   do: Next; 
    AndNot:        None               if +1:String == "^" {
        AsgnAndNot:       OpBitAsgnAndNot       if +2:String == "="   do: Next; 
        AndNot:           OpBitAndNot           if +1:String == "^"   do: Next; 
    }
    LogAnd:       OpLogAnd       if +1:String == "&"   do: Next; 
    BitAnd:       OpBitAnd       if String == "&"      do: Next; 
}
Or:		 None		 if String == "|" {
    AsgnOr:       OpBitAsgnOr       if +1:String == "="   do: Next; 
    LogOr:        OpLogOr           if +1:String == "|"   do: Next; 
    BitOr:        OpBitOr           if String == "|"      do: Next; 
}
// AnyText all lexers should end with a default AnyRune rule so lexing is robust 
AnyText:		 Text		 if AnyRune	 do: Next; 


///////////////////////////////////////////////////
// /Users/oreilly/goki/pi/langs/golang/go.pig Parser

// File only rules in this first group are used as top-level rules -- all others must be referenced from here 
File {
    PackageSpec:  'key:package' Name 'EOS'  >Ast
    Acts:{ -1:PushNewScope:"Name":NamePackage; -1:ChgToken:"Name":NamePackage; }
    Imports:  'key:import' ImportN 'EOS'  >Ast
    // Consts same as ConstDecl 
    Consts:  'key:const' ConstDeclN 'EOS'  >Ast
    // Types same as TypeDecl 
    Types:  'key:type' TypeDeclN 'EOS'  >Ast
    // Vars same as VarDecl 
    Vars:   'key:var' VarDeclN 'EOS'  >Ast
    Funcs:  @FunDecl 'EOS'            
    // Stmts this allows direct parsing of anything -- for one-line parsing 
    Stmts:  Stmt 'EOS'  
}
// ExprRules many different rules here that go into expressions etc 
ExprRules {
    // FullName name that is either a full package-qualified name or short plain name 
    FullName {
        // QualName package-qualified name 
        QualName:  'Name' '.' 'Name'  +Ast
        // Name just a name without package scope 
        Name:  'Name'  +Ast
    }
    // NameList one or more plain names, separated by , -- for var names 
    NameList {
        NameListEls:  @Name ',' NameList  >1Ast
        NameListEl:   'Name'              +Ast
    }
    ExprList {
        ExprListEls:  Expr ',' ExprList  
        ExprListEl:   Expr               
    }
    // Expr The full set of possible expressions 
    Expr {
        BinExpr:   BinaryExpr  
        UnryExpr:  UnaryExpr   
    }
    UnaryExpr {
        PosExpr:       '+' UnaryExpr  >Ast
        NegExpr:       '-' UnaryExpr  >Ast
        UnaryXorExpr:  '^' UnaryExpr  >Ast
        NotExpr:       '!' UnaryExpr  >Ast
        DePtrExpr:     '*' UnaryExpr  >Ast
        AddrExpr:      '&' UnaryExpr  >Ast
        // PrimExpr essential that this is LAST in unary list, so that distinctive first-position unary tokens match instead of more general cases in primary 
        PrimExpr:  PrimaryExpr  
    }
    // BinaryExpr due to top-down nature of parser, *lowest* precedence is *first* -- all rules *must* have - first = reverse order to get associativity right 
    BinaryExpr {
        NotEqExpr:       Expr '!=' Expr   >Ast
        EqExpr:          Expr '==' Expr   >Ast
        LogOrExpr:       Expr '||' Expr   >Ast
        LogAndExpr:      Expr '&&' Expr   >Ast
        GtEqExpr:        Expr '>=' Expr   >Ast
        GreaterExpr:     Expr '>' Expr    >Ast
        LtEqExpr:        Expr '<=' Expr   >Ast
        LessExpr:        Expr '<' Expr    >Ast
        BitOrExpr:       -Expr '|' Expr   >Ast
        BitXorExpr:      -Expr '^' Expr   >Ast
        BitAndExpr:      -Expr '&' Expr   >Ast
        ShiftRightExpr:  -Expr '>>' Expr  >Ast
        ShiftLeftExpr:   -Expr '<<' Expr  >Ast
        SubExpr:         -Expr '-' Expr   >Ast
        AddExpr:         -Expr '+' Expr   >Ast
        RemExpr:         -Expr '%' Expr   >Ast
        DivExpr:         -Expr '/' Expr   >Ast
        // MultExpr ! expr is exclusion conditions on '*' to deal with possibility of ptr type literal in map or slice 
        MultExpr:  -Expr '*' Expr ! ?'key:map' '[' ? ']' '*' 'Name' ?'.' ?'Name'  >Ast
    }
    PrimaryExpr {
        BasicLit:  BasicLiteral  
        // CompositeLit important to match sepcific '{' here, not using literal value -- must be before slice, to get map[] keyword instead of slice -- todo: had 'EOS' at the end -- not needed? 
        CompositeLit:  @LiteralType '{' ?ElementList ?'EOS' '}' ?PrimaryExpr      >Ast
        FuncLitCall:   'key:func' Signature '{' ?BlockList '}' '(' ?ArgsExpr ')'  >Ast
        FuncLit:       'key:func' Signature '{' ?BlockList '}'                    >Ast
        // ConvertBasic only works with basic builtin types -- others will get taken by FunCall 
        ConvertBasic:   @BasicType '(' Expr ')'                       >Ast
        ConvertParens:  '(' @Type ')' '(' Expr ?',' ')' ?PrimaryExpr  >Ast
        // Convert note: a regular type(expr) will be a FunCall 
        Convert:    @TypeLiteral '(' Expr ?',' ')'  >Ast
        ParenExpr:  '(' Expr ')' ?PrimaryExpr       
        // TypeAssert must be before FunCall to get . match 
        TypeAssert:  PrimaryExpr '.' '(' @Type ')' ?PrimaryExpr  >Ast
        // MakeCall takes type arg 
        MakeCall:  'key:make' '(' Type ?',' ?Expr ?',' ?Expr ')'  >Ast
        // NewCall takes type arg 
        NewCall:  'key:new' '(' Type ')'  >Ast
        // Selector This must be after unary expr esp addr, DePtr 
        Selector:  PrimaryExpr '.' PrimaryExpr  >Ast
        Acts:{ -1:ChgToken:"[0]":NameTag; }
        // Slice this needs further right recursion to keep matching more slices 
        Slice:     ?PrimaryExpr '[' SliceExpr ']' ?PrimaryExpr  >Ast
        MethCall:  ?PrimaryExpr '.' Name '(' ?ArgsExpr ')'      >Ast
        Acts:{ -1:ChgToken:"[0]":NameFunction; }
        // FuncCall must be after parens 
        FuncCall:  PrimaryExpr '(' ?ArgsExpr ')'  >Ast
        Acts:{ -1:ChgToken:"[0]":NameFunction; }
        // OpName this is the least selective and must be at the end 
        OpName:  FullName  
    }
    BasicLiteral {
        LitNumInteger:  'LitNumInteger'  +Ast
        LitNumFloat:    'LitNumFloat'    +Ast
        LitNumImag:     'LitNumImag'     +Ast
        // LitRune rune 
        LitRune:    'LitStrSingle'  +Ast
        LitString:  'LitStr'        +Ast
    }
    LiteralType {
        LitStructType:  StructType               
        LitIFaceType:   'key:interface' '{' '}'  +Ast
        // LitSliceType slice must come before array 
        LitSliceType:      SliceType           
        LitArrayAutoType:  ArrayAutoType       
        LitArrayType:      ArrayType           
        LitElType:         '[' '...' ']' Type  
        LitMapType:        MapType             
        // LitTypeName this is very general, must be at end.. 
        LitTypeName:  TypeName  
    }
    LiteralValue:  '{' ElementList ?'EOS' '}' 'EOS'  
    ElementList {
        ElementListEls:  KeyedEl ',' ?ElementList  
        KeyedEl {
            KeyEl:  Key ':' Element  >Ast
            Element {
                EmptyEl:   '{' '}'       _Ast
                ElExpr:    Expr          
                ElLitVal:  LiteralValue  
            }
        }
    }
    Key {
        KeyLitVal:  LiteralValue  
        KeyExpr:    Expr          
    }
    RecvType {
        RecvPtrType:    '(' '*' TypeName ')'  
        ParenRecvType:  '(' RecvType ')'      
        RecvTp:         TypeName              
    }
    Selectors {
        Sels:  Name '.' Selectors  
        Sel:   Name                
    }
    SubSlice:  '[' SliceExpr ']' ?SubSlice  _Ast
    SliceExpr {
        SliceThree:  ?SliceIdx1 ':' SliceIdx2 ':' SliceIdx3  >Ast
        SliceTwo:    ?SliceIdx1 ':' ?SliceIdx2               >Ast
        SliceOne:    Expr                                    >Ast
    }
    SliceIdxs {
        SliceIdx1:  Expr  >Ast
        SliceIdx2:  Expr  >Ast
        SliceIdx3:  Expr  >Ast
    }
    ArgsExpr {
        ArgsEllipsis:  ArgsList '...'  >Ast
        Args:          ArgsList        >Ast
    }
    ArgsList {
        ArgsListEls:  Expr ',' ArgsList  
        ArgsListEl:   Expr               
    }
}
TypeRules {
    // Type type specifies a type either as a type name or type expression 
    Type {
        ParenType:  '(' @Type ')'  
        TypeLit:    TypeLiteral    
        TypeNms:    TypeName       
    }
    TypeName {
        // BasicType recognizes builtin types 
        BasicType:  'KeywordType'  +Ast
        // QualType type equivalent to QualName 
        QualType:  'Name' '.' 'Name'  +Ast
        Acts:{ -1:ChgToken:"":NameType; }
        // TypeNm local unqualified type name 
        TypeNm:  'Name'  +Ast
        Acts:{ -1:ChgToken:"":NameType; }
    }
    // PtrOrTypeName regular type name or pointer to type name 
    PtrOrTypeName {
        PtrTypeName:    '*' TypeName  
        NoPtrTypeName:  TypeName      
    }
    TypeLiteral {
        SliceType:  '[' ']' @Type  >Ast
        Acts:{ 0:ChgToken:"../Name":NameArray; 0:AddSymbol:"../Name":NameArray; }
        // ArrayAutoType array must be after slice b/c slice matches on sequence of tokens 
        ArrayAutoType:  '[' '...' ']' @Type  >Ast
        Acts:{ 0:ChgToken:"../Name":NameArray; 0:AddSymbol:"../Name":NameArray; }
        // ArrayType array must be after slice b/c slice matches on sequence of tokens 
        ArrayType:  '[' Expr ']' @Type  >Ast
        Acts:{ 0:ChgToken:"../Name":NameArray; 0:AddSymbol:"../Name":NameArray; }
        StructType:  'key:struct' '{' ?FieldDecls '}' ?'EOS'  >Ast
        Acts:{ 0:ChgToken:"../Name":NameStruct; 0:PushNewScope:"../Name":NameStruct; -1:PopScope:"../Name":None; }
        PointerType:    '*' @Type                             >Ast
        FuncType:       'key:func' @Signature                 >Ast
        InterfaceType:  'key:interface' '{' ?MethodSpecs '}'  >Ast
        Acts:{ 0:ChgToken:"../Name":NameInterface; 0:PushNewScope:"../Name":NameInterface; -1:PopScope:"../Name":None; }
        MapType:  'key:map' '[' @Type ']' @Type  >Ast
        Acts:{ 0:ChgToken:"../Name":NameMap; 0:AddSymbol:"../Name":NameMap; }
        ChannelType {
            RecvChanType:  'key:chan' '<-' @Type  >Ast
            SendChanType:  '<-' 'key:chan' @Type  >Ast
            SRChanType:    'key:chan' @Type       >Ast
        }
    }
    FieldDecls:  FieldDecl ?FieldDecls  
    FieldDecl {
        AnonQualField:  'Name' '.' 'Name' ?FieldTag 'EOS'  >Ast
        Acts:{ -1:ChgToken:"":NamePackage; }
        NamedField:  NameList ?Type ?FieldTag 'EOS'  >Ast
        Acts:{ -1:ChgToken:"[0]":NameField; -1:AddSymbol:"[0]":NameField; }
    }
    FieldTag:  'LitStr'  +Ast
    // TypeDeclN N = switch between 1 or multi 
    TypeDeclN {
        TypeDeclGroup:  '(' TypeDecls ')'  
        TypeDeclEl:     Name Type 'EOS'    >Ast
        Acts:{ -1:ChgToken:"Name":NameType<-Name; -1:AddSymbol:"Name":None; -1:AddDetail:"[1]":None; -1:AddType:"Name":None; }
    }
    TypeDecls:  TypeDeclEl ?TypeDecls  
    TypeList {
        TypeListEls:  Type ',' TypeList  >1Ast
        TypeListEl:   Type               
    }
}
FuncRules {
    FunDecl {
        MethDecl:  'key:func' '(' MethRecv ')' Name Signature ?Block 'EOS'  >Ast
        Acts:{ 5:ChgToken:"Name":NameMethod; 5:PushNewScope:"Name":NameMethod; -1:AddDetail:"SigParams|SigParamsResult":None; -1:PopScope:"":None; -1:PopScope:"":None; }
        FuncDecl:  'key:func' Name Signature ?Block 'EOS'  >Ast
        Acts:{ -1:ChgToken:"Name":NameFunction; 2:PushNewScope:"Name":NameFunction; -1:AddDetail:"SigParams|SigParamsResult":None; -1:PopScope:"":None; }
    }
    MethRecv:  Name Type  >Ast
    Acts:{ -1:PushScope:"TypeNm|PointerType/TypeNm":NameStruct; }
    Signature {
        // SigParamsResult all types must fully match, using @ 
        SigParamsResult:  @Params @Result  
        SigParams:        @Params          >Ast
    }
    // MethodSpec for interfaces only -- interface methods 
    MethodSpec {
        MethSpecAnonQual:  'Name' '.' 'Name' 'EOS'  >Ast
        Acts:{ -1:ChgToken:"":NameInterface; -1:AddSymbol:"":NameInterface; }
        MethSpecName:  @Name @Params ?Result 'EOS'  >Ast
        Acts:{ -1:ChgToken:"Name":NameMethod; -1:AddSymbol:"Name":NameMethod; }
        MethSpecAnonLocal:  'Name' 'EOS'  >Ast
        Acts:{ -1:ChgToken:"":NameInterface; -1:AddSymbol:"":NameInterface; }
        MethSpecNone:  'EOS'  
    }
    MethodSpecs:  MethodSpec ?MethodSpecs  
    Result {
        Results:    '(' ParamsList ')'  
        ResultOne:  Type                
    }
    ParamsList {
        ParNameEllipsis:  ?ParamsList ?',' ?NameList '...' @Type  >Ast
        ParName:          @Name @Type ?',' ?ParamsList            _Ast
        ParType:          @Type ?',' ?ParamsList                  _Ast
        // ParNames need the explicit ',' in here to absorb so later one goes to paramslist 
        ParNames:  @Name ',' @NameList @Type ?',' ?ParamsList  _Ast
    }
    Params:  '(' ?ParamsList ')'  >Ast
}
StmtRules {
    StmtList:   Stmt 'EOS' ?StmtList  
    BlockList:  StmtList              >Ast
    Stmt {
        ConstDeclStmt:    'key:const' ConstDeclN 'EOS'  
        TypeDeclStmt:     'key:type' TypeDeclN 'EOS'    
        VarDeclStmt:      'key:var' VarDeclN 'EOS'      
        ReturnStmt:       'key:return' ?ExprList 'EOS'  >Ast
        BreakStmt:        'key:break' ?Name 'EOS'       >Ast
        ContStmt:         'key:continue' ?Name 'EOS'    >Ast
        GotoStmt:         'key:goto' Name 'EOS'         >Ast
        GoStmt:           'key:go' Expr 'EOS'           >Ast
        FallthroughStmt:  'key:fallthrough' 'EOS'       >Ast
        DeferStmt:        'key:defer' Expr 'EOS'        >Ast
        // IfStmt just matches if keyword 
        IfStmt {
            IfStmtExpr:  'key:if' Expr '{' ?BlockList '}' ?Elses 'EOS'                   >Ast
            IfStmtInit:  'key:if' SimpleStmt 'EOS' Expr '{' ?BlockList '}' ?Elses 'EOS'  >Ast
        }
        // ForStmt just for matching for token -- delegates to children 
        ForStmt {
            ForRangeExisting:  'key:for' ExprList '=' 'key:range' Expr '{' ?BlockList '}' 'EOS'   >Ast
            ForRangeNew:       'key:for' NameList ':=' 'key:range' Expr '{' ?BlockList '}' 'EOS'  >Ast
            Acts:{ -1:ChgToken:"NameListEls":NameVar; }
            ForRangeOnly:  'key:for' 'key:range' Expr '{' ?BlockList '}' 'EOS'  >Ast
            Acts:{ -1:ChgToken:"NameListEls":NameVar; }
            // ForExpr most general at end 
            ForExpr:  'key:for' ?Expr '{' ?BlockList '}' 'EOS'  >Ast
            // ForClauseStmt the embedded EOS's here require full expr here so final EOS has proper EOS StInc count 
            ForClauseStmt:  'key:for' ?SimpleStmt 'EOS' ?Expr 'EOS' ?PostStmt '{' ?BlockList '}' 'EOS'  >Ast
        }
        SwitchStmt {
            SwitchTypeName:  'key:switch' 'Name' ':=' PrimaryExpr '.' '(' 'key:type' ')' '{' BlockList '}' 'EOS'  >Ast
            Acts:{ 0:PushStack:"SwitchType":None; -1:PopStack:"":None; }
            SwitchTypeAnon:  'key:switch' PrimaryExpr '.' '(' 'key:type' ')' '{' BlockList '}' 'EOS'  >Ast
            Acts:{ 0:PushStack:"SwitchType":None; -1:PopStack:"":None; }
            SwitchExpr:          'key:switch' ?Expr '{' BlockList '}' 'EOS'                                                      >Ast
            SwitchInit:          'key:switch' SimpleStmt 'EOS' ?Expr '{' BlockList '}' 'EOS'                                     >Ast
            SwitchTypeNameInit:  'key:switch' SimpleStmt 'EOS' 'Name' ':=' PrimaryExpr '.' '(' Type ')' '{' BlockList '}' 'EOS'  >Ast
            SwitchTypeAnonInit:  'key:switch' SimpleStmt 'EOS' PrimaryExpr '.' '(' Type ')' '{' BlockList '}' 'EOS'              >Ast
        }
        CaseStmt {
            // TypeCaseEmptyStmt case and default require post-step to create sub-block -- no explicit { } scoping 
            TypeCaseEmptyStmt:  'key:case' @TypeList ':' 'EOS'  >Ast
            // TypeCaseStmt case and default require post-step to create sub-block -- no explicit { } scoping 
            TypeCaseStmt:  'key:case' @TypeList ':' Stmt  >Ast
            // CaseEmptyStmt case and default require post-step to create sub-block -- no explicit { } scoping 
            CaseEmptyStmt:  'key:case' ExprList ':' 'EOS'  >Ast
            // CaseExprStmt case and default require post-step to create sub-block -- no explicit { } scoping 
            CaseExprStmt:  'key:case' ExprList ':' Stmt  >Ast
            // SelCaseStmt case and default require post-step to create sub-block -- no explicit { } scoping 
            SelCaseStmt:  'key:case' CommStmt 'EOS' ?Stmt  >Ast
        }
        DefaultStmt:  'key:default' ':' ?Stmt  >Ast
        LabeledStmt:  @Name ':' ?Stmt          >Ast
        Acts:{ -1:ChgToken:"":NameLabel; }
        Block:     '{' ?StmtList '}' 'EOS'  >Ast
        SimpleSt:  SimpleStmt               
    }
    SimpleStmt {
        SendStmt:  ?Expr '<-' Expr 'EOS'  >Ast
        IncrStmt:  Expr '++' 'EOS'        >Ast
        DecrStmt:  Expr '--' 'EOS'        >Ast
        AsgnStmt:  Asgn                   
        ExprStmt:  Expr 'EOS'             >Ast
    }
    // PostStmt for loop post statement -- has no EOS 
    PostStmt {
        PostSendStmt:      ?Expr '<-' Expr                 >Ast
        PostIncrStmt:      Expr '++'                       >Ast
        PostDecrStmt:      Expr '--'                       >Ast
        PostAsgnExisting:  ExprList '=' ExprList           >Ast
        PostAsgnBit:       ExprList 'OpBitAsgn' ExprList   >Ast
        PostAsgnMath:      ExprList 'OpMathAsgn' ExprList  >Ast
        PostAsgnNew:       ExprList ':=' ExprList          >Ast
        Acts:{ -1:ChgToken:"Name...":NameVar<-Name; -1:AddSymbol:"Name":NameVar; -1:AddDetail:"[1]":None; }
        PostExprStmt:  Expr  >Ast
    }
    Asgn {
        AsgnExisting:  ExprList '=' ExprList 'EOS'   >Ast
        AsgnNew:       ExprList ':=' ExprList 'EOS'  >Ast
        Acts:{ -1:ChgToken:"Name...":NameVar<-Name; -1:AddSymbol:"Name":NameVar; -1:AddDetail:"[1]":None; }
        AsgnMath:  ExprList 'OpMathAsgn' ExprList 'EOS'  >Ast
        AsgnBit:   ExprList 'OpBitAsgn' ExprList 'EOS'   >Ast
    }
    Elses {
        ElseIfStmt:      'key:else' 'key:if' Expr '{' ?BlockList '}' ?Elses 'EOS'                   >Ast
        ElseStmt:        'key:else' '{' ?BlockList '}' 'EOS'                                        >Ast
        ElseIfStmtInit:  'key:else' 'key:if' SimpleStmt 'EOS' Expr '{' ?BlockList '}' ?Elses 'EOS'  >Ast
    }
    // CommStmt communication stmt: send or recv 
    CommStmt {
        CommSend:      SendStmt                  
        RecvExisting:  ExprList '=' Expr 'EOS'   >Ast
        RecvNew:       NameList ':=' Expr 'EOS'  >Ast
    }
}
ImportRules {
    // ImportN N = number switch (One vs. Group) 
    ImportN {
        // ImportGroup group of multiple imports 
        ImportGroup:  '(' ImportList ')'  
        // ImportOne single import -- ImportList also allows diff options 
        ImportOne:  ImportList  
    }
    ImportList {
        // ImportAlias put more specialized rules first 
        ImportAlias:  'Name' 'LitStr' ?'EOS' ?ImportList  +Ast
        Acts:{ -1:AddSymbol:"":NameLibrary; -1:ChgToken:"":NameLibrary; }
        Import:  'LitStr' ?'EOS' ?ImportList  +Ast
        Acts:{ -1:AddSymbol:"":NameLibrary; -1:ChgToken:"":NameLibrary; }
    }
}
DeclRules {
    TypeDecl:   'key:type' TypeDeclN 'EOS'    >Ast
    ConstDecl:  'key:const' ConstDeclN 'EOS'  
    VarDecl:    'key:var' VarDeclN 'EOS'      
    // ConstDeclN N = switch between 1 or group 
    ConstDeclN {
        ConstGroup:  '(' ConstList ')'  
        // ConstOpts different types of const expressions 
        ConstOpts {
            ConstSpec:  NameList ?Type '=' Expr 'EOS'  >Ast
            Acts:{ -1:ChgToken:"[0]":NameConstant; -1:AddSymbol:"[0]":NameConstant; -1:AddDetail:"[-1]":None; }
            // ConstSpecName only a name, no expression 
            ConstSpecName:  NameList 'EOS'  >Ast
            Acts:{ -1:ChgToken:"[0]":NameConstant; -1:AddSymbol:"[0]":NameConstant; }
        }
    }
    ConstList:  ConstOpts ?ConstList  
    // VarDeclN N = switch between 1 or group 
    VarDeclN {
        VarGroup:  '(' VarList ')'  
        // VarOpts different types of var expressions 
        VarOpts {
            VarSpecExpr:  NameList ?Type '=' Expr 'EOS'  >Ast
            Acts:{ -1:ChgToken:"[0]":NameVarGlobal; -1:AddSymbol:"[0]":NameVarGlobal; -1:AddDetail:"[-1]":None; }
            // VarSpec only a name and type, no expression 
            VarSpec:  NameList Type 'EOS'  >Ast
            Acts:{ -1:ChgToken:"[0]":NameVarGlobal; -1:AddSymbol:"[0]":NameVarGlobal; -1:AddDetail:"[1]":None; }
        }
    }
    VarList:  VarOpts ?VarList  
}
