package main

func AllowCookieMutation(m string, csrf, origin bool) bool {
	return (m == "POST" || m == "PUT" || m == "PATCH" || m == "DELETE") && csrf && origin
}
