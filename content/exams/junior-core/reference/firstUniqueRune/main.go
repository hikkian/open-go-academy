package main

func FirstUniqueRune(text string) string {
 runes:=[]rune(text); for i,r:=range runes { found:=false; for j,s:=range runes { if i!=j && r==s { found=true; break } }; if !found { return string(r) } }; return ""
}
