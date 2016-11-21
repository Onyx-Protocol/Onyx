This is the dockerfile that produces the docker image
we use for continuous integration.

To update it, do:

	TAG=chaindev/ci:`date +%Y%m%d`
	docker build --tag $TAG .
	docker push $TAG

To poke around in the image, do:

	docker run -ti --rm $TAG /bin/bash

Then edit $CHAIN/wercker.yml to use the new tag.
