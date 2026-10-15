package main

func MigrationVersionAllowed(version string) bool { return version == "0001_init" || version == "0002_link_index" }
