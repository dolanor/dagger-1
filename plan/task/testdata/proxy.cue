package main

import (
	"dagger.io/dagger"
	"universe.dagger.io/docker"
	"universe.dagger.io/bash"
)

dagger.#Plan & {
	actions: {
		pull: docker.#Pull & {
			source: "ubuntu:18.04"
		}

		display: bash.#Run & {
			input: pull.image
			script: contents: """
				ls /
				"""
		}
	}
}
