package main

type ProfileReaderEasy interface {
	FindProfile(id string) (name string, found bool)
}

func WelcomeEasy(repository ProfileReaderEasy, id string) string {
	if id == "" {
		return "invalid_request"
	}
	name, found := repository.FindProfile(id)
	if !found {
		return "not_found"
	}
	if name == "" {
		return "invalid_profile"
	}
	return "hello:" + name
}
