# Chain Core developer docs

#### Before you begin

Make sure you've installed `md2html`:

```
go install chain/cmd/md2html
```

#### Running the docs server

To host the docs on port 3001:

```
cd $CHAIN/doc/external/src
md2html :3001
```

#### Generating docs

To generate the docs as flat files:

```
cd $CHAIN/doc/external/src
md2html $CHAIN/doc/external/compiled
```

Note that `$CHAIN/doc/external/compiled` is in `.gitignore`, so it will not be committed to source control.
