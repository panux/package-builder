set -x
pref=x86_64-alpine-linux-musl-
for c in /usr/bin/$pref*; do
    d=${c/alpine/pc}
    ln -s $c ${d/musl/gnu}
done
