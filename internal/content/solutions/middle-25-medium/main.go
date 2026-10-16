package main

type ProfileRepositoryMedium interface {
	FindProfile(id string) (name string, found bool)
	SaveProfile(id, name string) bool
}

func RenameProfileMedium(repository ProfileRepositoryMedium, id, newName string) string {
	if id == "" || newName == "" {
		return "invalid_request"
	}
	if repository == nil {
		return "dependency_missing"
	}
	oldName, found := repository.FindProfile(id)
	if !found {
		return "not_found"
	}
	if oldName == "" {
		return "invalid_profile"
	}
	if oldName == newName {
		return "unchanged"
	}
	if !repository.SaveProfile(id, newName) {
		return "storage_error"
	}
	return "updated"
}
