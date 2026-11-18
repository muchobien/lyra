export MISE_DATA_DIR=/opt/mise
export MISE_CACHE_DIR=/opt/mise/cache
export MISE_YES=1
case ":$PATH:" in
*:/opt/mise/shims:*) ;;
*) export PATH="/opt/mise/shims:/opt/mise/bin:$HOME/.local/bin:$PATH" ;;
esac
if [ -r /run/orca/git/token ]; then
  GH_TOKEN="$(cat /run/orca/git/token)"
  export GH_TOKEN
fi
