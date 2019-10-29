echo "Downloading latest releases"
curl -s https://api.github.com/repos/nicolasgere/bobby/releases/latest \
| grep "bobby-osx" \
| cut -d : -f 2,3 \
| tr -d \" \
| wget -qi  - --show-progress -O bobby
echo "Make it executable"
chmod +x bobby
echo "Move it to /usr/bin/bobby"
mv bobby /usr/local/bin/bobby