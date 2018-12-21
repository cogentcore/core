// /Users/oreilly/goki/pi/langs/tex/tex.pig Lexer

Comment:		 Comment		 if String == "%"	 do: EOL; 
// LetterText optimization for plain letters which are always text 
LetterText:		 Text		 if Letter	 do: Next; 
// Backslash gets command after 
Backslash:		 NameBuiltin		 if String == "\"	 do: Next;  {
    Section:       NameBuiltin       if String == "section{"   do: Next;  {
        SectText:       TextStyleHeading       if AnyRune   do: ReadUntil: "}"; 
    }
    Subsection:       NameBuiltin       if String == "subsection{"   do: Next;  {
        SubSectText:       TextStyleSubheading       if AnyRune   do: ReadUntil: "}"; 
    }
    Subsubsection:       NameBuiltin       if String == "subsubsection{"   do: Next;  {
        SubSubSectText:       TextStyleSubheading       if AnyRune   do: ReadUntil: "}"; 
    }
    Bold:       NameBuiltin       if String == "textbf{"   do: Next;  {
        BoldText:       TextStyleStrong       if AnyRune   do: ReadUntil: "}"; 
    }
    Emph:       NameBuiltin       if String == "emph{"   do: Next;  {
        EmpText:       TextStyleEmph       if AnyRune   do: ReadUntil: "}"; 
    }
    TT:       NameBuiltin       if String == "textt{"   do: Next;  {
        TTText:       TextStyleOutput       if AnyRune   do: ReadUntil: "}"; 
    }
    VerbSlash:       NameBuiltin       if String == "verb\"   do: Next;  {
        VerbText:       TextStyleOutput       if AnyRune   do: ReadUntil: "\"; 
    }
    VerbPipe:       NameBuiltin       if String == "verb|"   do: Next;  {
        VerbText:       TextStyleOutput       if AnyRune   do: ReadUntil: ""; 
    }
    AnyCmd:       NameBuiltin       if AnyRune   do: Name; 
}
// LBraceBf old school.. 
LBraceBf:		 TextStyleStrong		 if String == "{\bf"	 do: ReadUntil: "}"; 
// LBraceEm old school.. 
LBraceEm:		 TextStyleEmph		 if String == "{\em"	 do: ReadUntil: "}"; 
LBrace:		 NameVar		 if String == "{"	 do: ReadUntil: "}"; 
LBrack:		 NameAttribute		 if String == "["	 do: ReadUntil: "]"; 
// RBrace straggler from prior special case 
RBrace:		 NameBuiltin		 if String == "}"	 do: Next; 
DollarSign:		 LitStr		 if String == "$"	 do: ReadUntil: "$"; 
Number:		 LitNum		 if @StartOfWord:Digit	 do: Number; 
Quotes:		 LitStrDouble		 if String == "``"	 do: ReadUntil: "''"; 
AnyText:		 Text		 if AnyRune	 do: Next; 


///////////////////////////////////////////////////
// /Users/oreilly/goki/pi/langs/tex/tex.pig Parser

