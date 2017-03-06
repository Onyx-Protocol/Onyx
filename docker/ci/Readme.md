This dockerfile produces the image used for continuous integration. Installed software includes:
- CentOS 6
- Go 1.8
- Java 1.7
- Ruby 2.0.0
- Node 4.8.0
- various build/test dependencies

To update it, do:

	TAG=chaindev/ci:`date +%Y%m%d`
	docker build --tag $TAG .
	docker push $TAG

Then edit $CHAIN/wercker.yml to use the new tag.

To install a dependency not found in a yum repository:
- Add an install script to the `bin` directory
- From inside the Dockerfile:
  - run the script (located in `/usr/bin`)
  - delete the script

To poke around in the image, do:

	docker run -ti --rm $TAG /bin/bash
