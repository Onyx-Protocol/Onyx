#!/bin/sh

# Clone Chain Core
git clone https://github.com/chain/chain.git $CHAIN
cd $CHAIN

# Install md2html
go install chain/cmd/md2html

# Build chain.com/docs
git checkout cmd.cored-1.1.0
cd $CHAIN/docs
md2html $DOCS_DEST/docs

# Build Java docs
echo
echo "Building Java SDK documentation..."

cd $CHAIN/sdk/java
git checkout sdk.java-1.1.0
mvn javadoc:javadoc

javadoc_dest_path=$DOCS_DEST/java
mkdir -p $javadoc_dest_path
cp -R target/site/apidocs/* $javadoc_dest_path

# Build Ruby docs
echo
echo "Building Ruby SDK documentation..."

cd $CHAIN/sdk/ruby
git checkout sdk.ruby-1.1.0
bundle
bundle exec yardoc 'lib/**/*.rb'

ruby_dest_path=$DOCS_DEST/ruby
mkdir -p $ruby_dest_path
cp -R doc/* $ruby_dest_path

# Build Node docs
echo
echo "Building Node.js SDK documentation..."

node_dest_path=$DOCS_DEST/node

cd $CHAIN/sdk/node
git checkout sdk.node-1.1.0
npm install
npm run docs
mkdir -p $node_dest_path
cp -R doc/* $node_dest_path
