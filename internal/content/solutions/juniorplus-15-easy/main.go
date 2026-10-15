package main

func PasswordKDFPolicy(i, s, k int) bool { return i >= 600000 && i <= 1200000 && s >= 16 && k == 32 }
