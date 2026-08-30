# THW OV Singen App - single command interface (casey/just)
# Docs site lives in ./docs (Docusaurus). Recipes below are the docs lifecycle.

set shell := ["bash", "-cu"]

# Install docs dependencies
docs-install:
    cd docs && npm install

# Start the Docusaurus dev server (default http://localhost:3000)
docs-start:
    cd docs && npm start

# Build the static docs site into docs/build
docs-build:
    cd docs && npm run build

# Build then serve the static site locally (validate production output)
docs-serve:
    cd docs && npm run build && npm run serve

# Clear Docusaurus caches
docs-clear:
    cd docs && npm run clear

# Alias: start the docs dev server
default:
    @just docs-start
