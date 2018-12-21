// /Users/oreilly/goki/pi/langs/tex/tex.pig Lexer

Comment:		 Comment		 if String == "%"	 do: EOL; 
// LetterText optimization for plain letters which are always text 
LetterText:		 Text		 if Letter	 do: Next; 
// Backslash gets command after 
Backslash:		 NameBuiltin		 if String == "\"	 do: Next;  {
    Section:             TextStyleHeading          if String == "section{"         do: EOL; 
    Subsection:          TextStyleSubheading       if String == "subsection{"      do: EOL; 
    Subsubsection:       TextStyleSubheading       if String == "subsubsection{"   do: ReadUntil; 
    Bold:                NameBuiltin               if String == "textbf{"          do: Next;  {
        BoldTxt:       TextStyleStrong       if AnyRune   do: ReadUntil; 
    }
    Emph:       NameBuiltin       if String == "emph{"   do: Next;  {
        EmpTxt:       TextStyleEmph       if AnyRune   do: ReadUntil; 
    }
    TT:       NameBuiltin       if String == "textt{"   do: Next;  {
        TTTxt:       TextStyleOutput       if AnyRune   do: ReadUntil; 
    }
    Verb:       NameBuiltin       if String == "verb\"   do: ReadUntil;  {
        // VerbTxt diff delims used here -- just supporting \ and | 
        VerbTxt:       TextStyleOutput       if AnyRune   do: ReadUntil; 
    }
    AnyCmd:       NameBuiltin       if AnyRune   do: Name; 
}
// LBraceBf old school.. 
LBraceBf:		 TextStyleStrong		 if String == "{\bf"	 do: ReadUntil; 
// LBraceEm old school.. 
LBraceEm:		 TextStyleEmph		 if String == "{\em"	 do: ReadUntil; 
LBrace:		 NameFunction		 if String == "{"	 do: ReadUntil; 
LBrack:		 NameAttribute		 if String == "["	 do: ReadUntil; 
// RBrace straggler from prior special case 
RBrace:		 NameBuiltin		 if String == "}"	 do: Next; 
DollarSign:		 LitStr		 if String == "$"	 do: ReadUntil; 
Number:		 LitNum		 if Digit	 do: Number; 
AnyText:		 Text		 if AnyRune	 do: Next; 


///////////////////////////////////////////////////
// /Users/oreilly/goki/pi/langs/tex/tex.pig Parser

