#!/bin/sh
# Saves the pages of real-urls.txt in real/, as the HTML that htmlcore
# gets with its default GetURL (as cogent/web does), with a .url file for each page with the
# URL that its relative URLs are relative to (after redirects).
cd "$(dirname "$0")"
mkdir -p real
grep -v '^#' real-urls.txt | while read -r name url; do
	[ -z "$name" ] && continue
	final=$(curl -sL -A "Mozilla/5.0 (compatible; CogentCore; +https://cogentcore.org)" -o "real/$name.html" -w '%{url_effective}' "$url")
	echo "$final" > "real/$name.url"
	echo "$name $(wc -c < "real/$name.html") bytes $final"
done
