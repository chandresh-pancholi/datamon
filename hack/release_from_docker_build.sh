#! /bin/sh

STAGE_DIR=/tmp/datamon-release
REL_DIR=out/release

if [ -d "$STAGE_DIR" ]; then
    rm -rf "$STAGE_DIR"
fi

if [ -d "$REL_DIR" ]; then
    rm -rf "$REL_DIR"
fi

mkdir -p "$STAGE_DIR"
mkdir -p "$REL_DIR"

docker_container=$(docker create datamon-binaries)

for plat in linux mac; do
    mkdir "${STAGE_DIR}/${plat}"
    docker cp "$docker_container:/stage/usr/bin/datamon.${plat}" "${STAGE_DIR}/${plat}/datamon"
    (cd "${STAGE_DIR}/${plat}" && \
         tar -cvzf "datamon.${plat}.tgz" datamon)
    mv "${STAGE_DIR}/${plat}/datamon.${plat}.tgz" "$REL_DIR"
done

(cd "$REL_DIR" && \
     for archive in $(ls -1 datamon.*.tgz); do
         md5sum $archive >> datamon.dsc
     done)
