package service

const octoModuleDaemonOnly = true

func isOctoModuleDaemonOnly() bool {
	return octoModuleDaemonOnly
}

func octoModuleDaemonOnlyError(action string) error {
	message := "octomodule is daemon-only"
	if trim(action) != "" {
		message = message + "; " + trim(action) + " is not supported"
	}
	return invalidInput(message)
}
