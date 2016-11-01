# Chain Core app for macOS

## Build instructions

Make sure you are running macOS Sierra (10.12) and have Xcode 8. 

1) Install [MacPorts](https://www.macports.org/install.php).

2) Install depedencies necessary to build Postgres:

    $ sudo port install autoconf automake json-c docbook-dsssl docbook-sgml-4.2 docbook-xml-4.2 docbook-xsl libxslt openjade opensp

3a) Install iTerm in order to be able to make `ClientLauncher.applescript` compile.

3b) Comment out the entire `open_iTerm` block of code in `ClientLauncher.applescript` if you do not want to install iTerm.

4) Build Postgres (binaries will be installed in `./pg/build`):

    $ cd pg/src
    $ make clean
    $ make -j16

5) Open ChainCore.xcodeproj and click Build.

Note: `make` patches the absolute loader paths into relative ones which messes with postgres's configure script.
Make sure you do `make clean` before re-build. Later we will improve this by patching separate copies of the libraries, during cope_postgres.sh phase.


## Release process

1. Bump the APP_VERSION in project settings.
2. Bump the DB_VERSION in project settings if necessary (this will make the app create separate database and ignore all previous data).
3. Build and Archive the app.
4. Export with Developer ID signature to the `updates` folder.
5. Place `Chain Core.app` in the `updates` folder directly, without extra stuff.
6. Run `tools/update_appcast.rb`. 
7. Edit `updates/updates.xml` to specify relevant release notes.
8. Upload the latest `Chain_Core_X.Y.zip`, `Chain_Core.zip`, and `updates.xml` to the server specified in the `tools/update_appcast.rb`. 


## License

Copyright © 2016 Chain, Inc.

Visit the following link for more info on third-party software licenses:
https://chain.com/docs/core/reference/license
