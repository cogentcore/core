// /Users/oreilly/goki/pi/langs/markdown/markdown.pig Lexer

InCode:		 None		 if CurState == "Code" {
    CodeEnd:       LitStrBacktick       if String == "```"   do: PopGuestLex; PopState; Next; 
    AnyCode:       LitStrBacktick       if AnyRune           do: Next; 
}
InLinkAddr:		 None		 if CurState == "LinkAddr" {
    EndLinkAddr:       NameAttribute       if String == ")"   do: PopState; Next; 
    AnyLinkAddr:       NameAttribute       if AnyRune         do: Next; 
}
InLinkTag:		 None		 if CurState == "LinkTag" {
    LinkAddr:       NameAttribute       if +1:String == "("   do: PopState; PushState: LinkAddr; Next; 
    // EndLinkTag for a plain tag with no addr 
    EndLinkTag:       NameTag       if String == "]"   do: PopState; Next; 
    AnyLinkTag:       NameTag       if AnyRune         do: Next; 
}
InBoldStars:		 None		 if CurState == "BoldStars" {
    EndBoldStars:       TextStyleStrong       if String == "**"   do: PopState; Next; 
    AnyBoldStars:       TextStyleStrong       if AnyRune          do: Next; 
}
InBoldUnders:		 None		 if CurState == "BoldUnders" {
    EndBoldUnders:       TextStyleStrong       if String == "__"   do: PopState; Next; 
    AnyBoldUnders:       TextStyleStrong       if AnyRune          do: Next; 
}
InEmphStar:		 None		 if CurState == "EmphStar" {
    EndEmphStar:       TextStyleEmph       if String == "*"   do: PopState; Next; 
    AnyEmphStar:       TextStyleEmph       if AnyRune         do: Next; 
}
InEmphUnder:		 None		 if CurState == "EmphUnder" {
    EndEmphUnder:       TextStyleEmph       if String == "_"   do: PopState; Next; 
    AnyEmphUnder:       TextStyleEmph       if AnyRune         do: Next; 
}
CodeStart:		 LitStrBacktick		 if String == "```"	 do: Next; PushState: Code;  {
    CodeLang:        KeywordNamespace       if Letter    do: Name; SetGuestLex; 
    CodePlain:       LitStrBacktick         if AnyRune   do: Next; 
}
HeadPound:		 None		 if String == "#" {
    HeadPound2:       None       if +1:String == "#" {
        HeadPound3:       None       if +2:String == "#" {
            SubSubHeading:       TextStyleSubheading       if +3:AnyRune   do: EOL; 
        }
        SubHeading:       TextStyleSubheading       if +2:WhiteSpace   do: EOL; 
    }
    Heading:       TextStyleHeading       if +1:WhiteSpace   do: EOL; 
}
ItemCheck:		 KeywordType		 if String == "- [" {
    ItemCheckDone:       KeywordType         if String == "- [x] "   do: Next; 
    ItemCheckTodo:       NameException       if String == "- [ ] "   do: Next; 
}
ItemStar:		 Keyword		 if String == "*"	 do: Next; 
ItemPlus:		 Keyword		 if String == "+"	 do: Next; 
ItemMinus:		 Keyword		 if String == "-"	 do: Next; 
BoldStars:		 TextStyleStrong		 if String == "**"	 do: PushState: BoldStars; Next; 
BoldUnders:		 TextStyleStrong		 if String == "__"	 do: PushState: BoldUnders; Next; 
ItemStarSub:		 Keyword		 if +4:String == "*"	 do: Next; 
ItemPlusSub:		 Keyword		 if +4:String == "+"	 do: Next; 
ItemMinusSub:		 Keyword		 if +4:String == "-"	 do: Next; 
LinkTag:		 NameTag		 if String == "["	 do: PushState: LinkTag; Next; 
BacktickCode:		 LitStrBacktick		 if String == "`"	 do: QuotedRaw; 
Quote:		 LitStrDouble		 if String == """	 do: QuotedRaw; 
Apostrophe:		 LitStrSingle		 if String == "'" {
    QuoteSingle:       LitStrSingle       if +2:String == "'"   do: Next; 
    Apost:             None               if String == "'"      do: Next; 
}
EmphStar:		 TextStyleEmph		 if String == "*"	 do: PushState: EmphStar; Next; 
EmphUnder:		 TextStyleEmph		 if String == "_"	 do: PushState: EmphUnder; Next; 
SkipWhite:		 TextWhitespace		 if WhiteSpace	 do: Next; 
AnyText:		 Text		 if AnyRune	 do: Next; 


///////////////////////////////////////////////////
// /Users/oreilly/goki/pi/langs/markdown/markdown.pig Parser

