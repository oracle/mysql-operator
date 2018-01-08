STASH_NAME=pre-commit-$(date +%s)
git stash save -q --keep-index $STASH_NAME

DOCKER_OPS_INTERACTIVE=-t make test
RESULT=$?

STASH_NUM=$(git stash list | grep $STASH_NAME | sed -re 's/stash@\{(.*)\}.*/\1/')

if [ -n "$STASH_NUM" ]; then
    git stash pop -q stash@{$STASH_NUM}
fi

[ $RESULT -ne 0 ] && exit 1
exit 0
