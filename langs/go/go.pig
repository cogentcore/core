// /Users/oreilly/goki/pi/langs/go/go.pig Lexer

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
        iota:              Keyword       if StrName == "iota"          do: Name; 
        map:               Keyword       if StrName == "map"           do: Name; 
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
        string:           KeywordType       if StrName == "string"       do: Name; 
        int:              KeywordType       if StrName == "int"          do: Name; 
        int8:             KeywordType       if StrName == "int8"         do: Name; 
        int16:            KeywordType       if StrName == "int16"        do: Name; 
        int32:            KeywordType       if StrName == "int32"        do: Name; 
        int64:            KeywordType       if StrName == "int64"        do: Name; 
        uint:             KeywordType       if StrName == "uint"         do: Name; 
        uint8:            KeywordType       if StrName == "uint8"        do: Name; 
        uint16:           KeywordType       if StrName == "uint16"       do: Name; 
        uint32:           KeywordType       if StrName == "uint32"       do: Name; 
        uint64:           KeywordType       if StrName == "uint64"       do: Name; 
        uintptr:          KeywordType       if StrName == "uintptr"      do: Name; 
        byte:             KeywordType       if StrName == "byte"         do: Name; 
        rune:             KeywordType       if StrName == "rune"         do: Name; 
        float32:          KeywordType       if StrName == "float32"      do: Name; 
        float64:          KeywordType       if StrName == "float64"      do: Name; 
        complex64:        KeywordType       if StrName == "complex64"    do: Name; 
        complex128:       KeywordType       if StrName == "complex128"   do: Name; 
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
LitStrSingle:		 LitStrSingle		 if String == "'"	 do: EOL; 
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
    Not:            OpAsgnAssign        if String == "!"      do: Next; 
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


///////////////////////////////////////////////////
// /Users/oreilly/goki/pi/langs/go/go.pig Parser

// File only rules in this first group are used as top-level rules -- all others must be referenced from here 
File {
    PackageSpec:  'key:package' Name 'EOS'      
    Imports:      'key:import' ImportN 'EOS'    
    Consts:       'key:const' ConstDeclN 'EOS'  
    Types:        'key:type' TypeDeclN 'EOS'    
    Vars:         'key:var' VarDeclN 'EOS'      
    Funcs:        FunDecl 'EOS'                 
}
// ExprRules many different rules here that go into expressions etc 
ExprRules {
    // FullName name that is either a full package-qualified name or short plain name 
    FullName {
        // QualName package-qualified name 
        QualName:  'Name' '.' 'Name'  
        // Name just a name without package scope 
        Name:  'Name'  
    }
    // NameList one or more plain names, separated by , -- for var names 
    NameList {
        NameListMulti:  'Name' ',' NameList  
        NameOne:        'Name'               
    }
    ExprList {
        ExprListMulti:  Expr ',' ExprList  
        ExprOne:        Expr               
    }
    // Expr The full set of possible expressions 
    Expr {
        BinExpr:   BinaryExpr  
        UnryExpr:  UnaryExpr   
    }
    UnaryExpr {
        PosExpr:       '+' @UnaryExpr  
        NegExpr:       '-' @UnaryExpr  
        UnaryXorExpr:  '^' @UnaryExpr  
        NotExpr:       '!' @UnaryExpr  
        DePtrExpr:     '*' @UnaryExpr  
        AddrExpr:      '&' @UnaryExpr  
        // PrimExpr essential that this is LAST in unary list, so that distinctive first-position unary tokens match instead of more general cases in primary 
        PrimExpr:  PrimaryExpr  
    }
    // BinaryExpr due to top-down nature of parser, *lowest* precedence is *first* -- all rules *must* have - first = reverse order to get associativity right 
    BinaryExpr {
        LogOrExpr:       -Expr '||' Expr  
        LogAndExpr:      -Expr '&&' Expr  
        BitOrExpr:       -Expr '|' Expr   
        BitXorExpr:      -Expr '^' Expr   
        BitAndExpr:      -Expr '&' Expr   
        NotEqExpr:       -Expr '!=' Expr  
        EqExpr:          -Expr '==' Expr  
        GtEqExpr:        -Expr '>=' Expr  
        GreaterExpr:     -Expr '>' Expr   
        LtEqExpr:        -Expr '<=' Expr  
        LessExpr:        -Expr '<' Expr   
        ShiftRightExpr:  -Expr '>>' Expr  
        ShiftLeftExpr:   -Expr '<<' Expr  
        SubExpr:         -Expr '-' Expr   
        AddExpr:         -Expr '+' Expr   
        RemExpr:         -Expr '%' Expr   
        DivExpr:         -Expr '/' Expr   
        MultExpr:        -Expr '*' Expr   
    }
    PrimaryExpr {
        BasicLit:  BasicLiteral  
        // CompositeLit important to match sepcific '{' here, not using literal value -- must be before slice, to get map[] keyword instead of slice 
        CompositeLit:  LiteralType '{' ElementList '}' 'EOS'  
        // ConvertBasic only works with basic builtin types -- others will get taken by FunCall 
        ConvertBasic:   @BasicType '(' @Expr ')'          
        ConvertParens:  '(' @Type ')' '(' @Expr ?',' ')'  
        // Convert note: a regular type(expr) will be a FunCall 
        Convert:    @TypeLiteral '(' Expr ?',' ')'        
        ParenExpr:  '(' Expr ')'                          
        MethCall:   @RecvType '.' Name '(' ?ArgsExpr ')'  
        // TypeAssert must be before FunCall to get . match 
        TypeAssert:  PrimaryExpr '.' '(' @Type ')'  
        // FuncCall must be after parens 
        FuncCall:  PrimaryExpr '(' ?ArgsExpr ')'  
        Selector:  PrimaryExpr '.' Name           
        Slice:     PrimaryExpr '[' SliceExpr ']'  
        // OpName this is the least selective and must be at the end 
        OpName:  FullName  
    }
    BasicLiteral {
        LitNumInteger:  'LitNumInteger'  
        LitNumFloat:    'LitNumFloat'    
        LitNumImag:     'LitNumImag'     
        // LitRune rune 
        LitRune:  'LitStrSingle'  
        // LitString rune 
        LitString:  'LitStr'  
    }
    LiteralType {
        LitStructType:  StructType  
        // LitSliceType slice must come before array 
        LitSliceType:  SliceType           
        LitArrayType:  ArrayType           
        LitElType:     '[' '...' ']' Type  
        LitMapType:    MapType             
        // LitTypeName this is very general, must be at end.. 
        LitTypeName:  TypeName  
    }
    LiteralValue:  '{' ElementList '}' 'EOS'  
    ElementList {
        ElementListMulti:  KeyedElement ',' ?ElementList  
        KeyedElement {
            KeyElement:  Key ':' Element  
            Element {
                ElExpr:    Expr          
                ElLitVal:  LiteralValue  
            }
        }
    }
    Key {
        KeyName:    'Name'        
        KeyLitVal:  LiteralValue  
        KeyExpr:    Expr          
    }
    RecvType {
        RecvPtrType:    '(' '*' TypeName ')'  
        ParenRecvType:  '(' RecvType ')'      
        RecvTp:         TypeName              
    }
    SliceExpr {
        SliceThree:  ?SliceIdx1 ':' SliceIdx2 ':' SliceIdx3  
        SliceTwo:    ?SliceIdx1 ':' ?SliceIdx2               
        SliceOne:    Expr                                    
    }
    SliceIdxs {
        SliceIdx1:  Expr  
        SliceIdx2:  Expr  
        SliceIdx3:  Expr  
    }
    ArgsExpr {
        ArgsEllipsis:  ExprList '...'  
        Args:          ExprList        
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
        BasicType:  'KeywordType'  
        // QualType type equivalent to QualName 
        QualType:  'Name' '.' 'Name'  
        // TypeNm local unqualified type name 
        TypeNm:  'Name'  
    }
    // PtrOrTypeName regular type name or pointer to type name 
    PtrOrTypeName {
        PtrTypeName:    '*' TypeName  
        NoPtrTypeName:  TypeName      
    }
    TypeLiteral {
        SliceType:  '[' ']' @Type  
        // ArrayType array must be after slice b/c slice matches on sequence of tokens 
        ArrayType:      '[' @Expr ']' @Type                      
        StructType:     'key:struct' '{' ?FieldDecls '}' ?'EOS'  
        PointerType:    '*' @Type                                
        FuncType:       'key:func' @Signature                    
        InterfaceType:  'key:interface' '{' MethodSpecs '}'      
        MapType:        'key:map' '[' @Type ']' @Type            
        ChannelType {
            RecvChanType:  'key:chan' '<-' @Type  
            SendChanType:  '<-' 'key:chan' @Type  
            SRChanType:    'key:chan' @Type       
        }
    }
    FieldDecls:  FieldDecl ?FieldDecls  
    FieldDecl {
        NamedField:  NameList Type ?FieldTag 'EOS'  
        AnonField:   PtrOrTypeName ?FieldTag 'EOS'  
    }
    FieldTag:  'LitStr'  
    // TypeDeclN N = switch between 1 or multi 
    TypeDeclN {
        TypeDeclMulti:  '(' TypeDecls ')'  
        TypeDecl:       Name Type 'EOS'    
    }
    TypeDecls:  TypeDecl ?TypeDecls  
}
FuncRules {
    FunDecl {
        MethDecl:  'key:func' '(' Name Type ')' Name Signature ?Block 'EOS'  
        FuncDecl:  'key:func' Name Signature ?Block 'EOS'                    
    }
    Signature:  Params ?Result  
    // MethodSpec for interfaces only -- interface methods 
    MethodSpec {
        MethSpecName:  'Name' Params ?Result 'EOS'  
        MethSpecAnon:  TypeName                     
    }
    MethodSpecs:  MethodSpec ?MethodSpecs  
    Result {
        Results:    '(' ParamsList ')'  
        ResultOne:  Type                
    }
    Param {
        ParamNameEllipsis:  ?NameList '...' Type  
    }
    ParamsList {
        ParName:  @Name @Type ?',' ?ParamsList  
        // ParNames need the explicit ',' in here to absorb so later one goes to paramslist 
        ParNames:  Name ',' @NameList @Type ?',' ?ParamsList  
        ParType:   @Type ?',' ?ParamsList                     
    }
    Params:  '(' ?ParamsList ')'  
}
StmtRules {
    StmtList:   Stmt 'EOS' ?StmtList  
    BlockList:  StmtList              
    Stmt {
        ReturnStmt:       'key:return' ?ExprList 'EOS'                                    
        BreakStmt:        'key:break' ?Name 'EOS'                                         
        ContStmt:         'key:continue' ?Name 'EOS'                                      
        GotoStmt:         'key:goto' Name 'EOS'                                           
        FallthroughStmt:  'key:fallthrough' 'EOS'                                         
        DeferStmt:        'key:defer' Expr 'EOS'                                          
        IfStmtInit:       'key:if' SimpleStmt 'EOS' Expr '{' ?BlockList '}' ?Elses 'EOS'  
        IfStmtExpr:       'key:if' Expr '{' ?BlockList '}' ?Elses 'EOS'                   
        // ForClauseStmt the embedded EOS's here require full expr here so final EOS has proper EOS StInc count 
        ForClauseStmt:     'key:for' ?SimpleStmt 'EOS' ?Expr 'EOS' ?SimpleStmt '{' ?BlockList '}' 'EOS'  
        ForRangeExisting:  'key:for' ExprList '=' 'key:range' Expr '{' ?BlockList '}' 'EOS'              
        ForRangeNew:       'key:for' NameList ':=' 'key:range' Expr '{' ?BlockList '}' 'EOS'             
        // ForExpr most general at end 
        ForExpr:     'key:for' ?Expr '{' ?BlockList '}' 'EOS'                     
        SwitchInit:  'key:switch' SimpleStmt 'EOS' ?Expr '{' BlockList '}' 'EOS'  
        SwitchExpr:  'key:switch' ?Expr '{' BlockList '}' 'EOS'                   
        // CaseStmt case and default require post-step to create sub-block -- no explicit { } scoping 
        CaseStmt:     'key:case' ExprList ':' ?Stmt  
        DefaultStmt:  'key:default' ':' ?Stmt        
        LabeledStmt:  Name ':' Stmt                  
        Block:        '{' ?StmtList '}' 'EOS'        
        SimpleSt:     SimpleStmt                     
    }
    SimpleStmt {
        SendStmt:  Expr '<-' Expr 'EOS'  
        IncrStmt:  Expr '++' 'EOS'       
        DecrStmt:  Expr '--' 'EOS'       
        AsgnStmt:  Asgn                  
        ExprStmt:  Expr 'EOS'            
    }
    Asgn {
        AsgnExisting:  ExprList '=' ExprList 'EOS'           
        AsgnNew:       ExprList ':=' ExprList 'EOS'          
        AsgnMath:      ExprList 'OpMathAsgn' ExprList 'EOS'  
        AsgnBit:       ExprList 'OpBitAsgn' ExprList 'EOS'   
    }
    Elses {
        ElseIfStmt:  'key:else' 'key:if' Expr '{' ?BlockList '}' ?Elses 'EOS'  
        ElseStmt:    'key:else' '{' ?BlockList '}' 'EOS'                       
    }
}
ImportRules {
    // ImportN N = number switch (One vs. Multi) 
    ImportN {
        // ImportMulti multiple imports 
        ImportMulti:  '(' ImportList ')'  
        // ImportOne single import -- ImportList also allows diff options 
        ImportOne:  ImportList  
    }
    ImportList {
        // ImportSpecAlias put more specialized rules first 
        ImportSpecAlias:  'Name' 'LitStr' ?'EOS' ?ImportList  
        ImportSpec:       'LitStr' ?'EOS' ?ImportList         
    }
}
DeclRules {
    // ConstDeclN N = switch between 1 or multi 
    ConstDeclN {
        ConstMulti:  '(' ConstList ')'  
        // ConstOpts different types of const expressions 
        ConstOpts {
            ConstSpec:  NameList ?Type '=' Expr 'EOS'  
            // ConstSpecName only a name, no expression 
            ConstSpecName:  NameList 'EOS'  
        }
    }
    ConstList:  ConstOpts ?ConstList  
    // VarDeclN N = switch between 1 or multi 
    VarDeclN {
        VarMulti:  '(' VarList ')'  
        // VarOpts different types of var expressions 
        VarOpts {
            VarSpecExpr:  NameList ?Type '=' Expr 'EOS'  
            // VarSpec only a name and type, no expression 
            VarSpec:  NameList Type 'EOS'  
        }
    }
    VarList:  VarOpts ?VarList  
}
