## patches

This file records small changes we've made to vendored dependencies. If you are going to update a dependency listed here, please be conscientious about the patches we've made.

In the future we will likely have a more sophisticated patching system.

### github.com/google/snappy:
at 7b9532b8781c5beaf99eef8ca7535d969a129d78, changes CMakeLists.txt to build a `STATIC` library instead of a `SHARED` library: https://github.com/google/snappy/blob/7b9532b8781c5beaf99eef8ca7535d969a129d78/CMakeLists.txt#L84