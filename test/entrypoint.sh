#!/usr/bin/env sh
set -e

help() {
  echo "This entrypoint limit users input to make targets but provides the possibility"
  echo "to clone a repository before executing make. These are the accepted args:"
  echo ""
  echo "/entrypoint.sh [-v|--verbose] [-g|--github|--repository]  [-r|--ref|--branch|--commit]  [-h|--help] [--] <make target and variables>"
  echo ""
  echo "  -v, --verbose         Set shell's verbose mode"
  echo "  -g, --repository      Set repository's url from where to clone it"
  echo "  -r, --ref, --commit   Reference to set in this task run"
  echo "  -h, --help            Show this help message"
  exit 1
}

while true; do
  case "$1" in
    -v | --verbose ) set -x; echo "Called with these arguments: $*"; shift ;;
    -g | --repository ) REPO="$2"; shift 2 ;;
    -r | --ref | --commit ) REF="$2"; shift 2 ;;
    -h | --help ) help ;;
    -- ) shift; break ;;
    * ) break ;;
  esac
done

echo "Cloning repository '${REPO}'"
git clone --progress --no-checkout -- "${REPO}" working-directory

cd working-directory

echo "Checking out '${REF}'"
git checkout "${REF}"

echo "Calling: make $*"
make "$@"
