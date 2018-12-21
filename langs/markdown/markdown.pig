// /Users/oreilly/goki/pi/langs/markdown/markdown.pig Lexer

InCode:		 None		 if CurState == "Code" {
    CodeEnd:       LitStrBacktick       if String == "```"   do: PopGuestLex; PopState; Next; 
    AnyCode:       LitStrBacktick       if AnyRune           do: Next; 
}
InLinkAttr:		 None		 if CurState == "LinkAttr" {
    EndLinkAttr:       NameVar       if String == "}"   do: PopState; Next; 
    AnyLinkAttr:       NameVar       if AnyRune         do: Next; 
}
InLinkAddr:		 None		 if CurState == "LinkAddr" {
    LinkAttr:          NameAttribute       if String == "){"   do: PopState; PushState: LinkAttr; Next; 
    EndLinkAddr:       NameAttribute       if String == ")"    do: PopState; Next; 
    AnyLinkAddr:       NameAttribute       if AnyRune          do: Next; 
}
InLinkTag:		 None		 if CurState == "LinkTag" {
    LinkAddr:       NameTag       if String == "]("   do: PopState; PushState: LinkAddr; Next; 
    // EndLinkTag for a plain tag with no addr 
    EndLinkTag:       NameTag       if String == "]"   do: PopState; Next; 
    AnyLinkTag:       NameTag       if AnyRune         do: Next; 
}
// LetterText optimization for plain letters which are always text 
LetterText:		 Text		 if Letter	 do: Next; 
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
// ItemStar note: these all have a space after them! 
ItemStar:		 Keyword		 if String == "* "	 do: Next; 
ItemPlus:		 Keyword		 if String == "+ "	 do: Next; 
ItemMinus:		 Keyword		 if String == "- "	 do: Next; 
NumList:		 Keyword		 if Digit	 do: Next; 
CommentStart:		 Comment		 if String == "<!---"	 do: ReadUntil; 
QuotePara:		 TextStyleUnderline		 if String == "> "	 do: EOL; 
BoldStars:		 TextStyleStrong		 if String == " **"	 do: Next;  {
    BoldText:       TextStyleStrong       if String == ""   do: ReadUntil; 
}
BoldUnders:		 TextStyleStrong		 if String == " __"	 do: Next;  {
    BoldText:       TextStyleStrong       if String == ""   do: ReadUntil; 
}
// ItemStarSub note all have space after 
ItemStarSub:		 Keyword		 if +4:String == "* "	 do: Next; 
ItemPlusSub:		 Keyword		 if +4:String == "+ "	 do: Next; 
ItemMinusSub:		 Keyword		 if +4:String == "- "	 do: Next; 
LinkTag:		 NameTag		 if String == "["	 do: PushState: LinkTag; Next; 
BacktickCode:		 LitStrBacktick		 if String == "`"	 do: QuotedRaw; 
Quote:		 LitStrDouble		 if String == """	 do: QuotedRaw; 
Apostrophe:		 LitStrSingle		 if String == "'" {
    QuoteSingle:       LitStrSingle       if +2:String == "'"   do: Next; 
    Apost:             None               if String == "'"      do: Next; 
}
EmphStar:		 TextStyleEmph		 if String == " *"	 do: Next;  {
    EmphText:       TextStyleEmph       if String == ""   do: ReadUntil; 
}
EmphUnder:		 TextStyleEmph		 if String == " _"	 do: Next;  {
    EmphUnder:       TextStyleEmph       if String == ""   do: ReadUntil; 
}
AnyText:		 Text		 if AnyRune	 do: Next; 


///////////////////////////////////////////////////
// /Users/oreilly/goki/pi/langs/markdown/markdown.pig Parser

