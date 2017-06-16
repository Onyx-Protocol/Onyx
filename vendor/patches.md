## patches

This file records small changes we've made to vendored dependencies. If you are going to update a dependency listed here, please be conscientious about the patches we've made.

In the future we will likely have a more sophisticated patching system.

### github.com/google/snappy:
 at 7b9532b8781c5beaf99eef8ca7535d969a129d78:

 * changes CMakeLists.txt to build a `STATIC` library instead of a `SHARED` library: https://github.com/google/snappy/blob/7b9532b8781c5beaf99eef8ca7535d969a129d78/CMakeLists.txt#L84
 * adds gitignore to ignore the `build` directory


### github.com/tecbot/gorocksdb
at 209fbe7c598c4e5b6da30d90fa1e70f80bb887d7, changes dynflag.go to use static libraries for snappy and rocksdb