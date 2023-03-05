# gget
Simple CLI utility to download files with visual progress information and time remaining estimation. 

## How To Use
```bash
# first build it
sudo CGO_ENABLED=0 go build -o /usr/local/bin/gget .

# now start downloading, like so:

# download test file to temp dir
gget https://proof.ovh.net/files/100Mb.dat /tmp/

# download test file to current dir
gget https://proof.ovh.net/files/100Mb.dat
```
## Example Output
![Example Output](example.png?raw=true "Example Output")

## Existing Files
If a file with the same name as the download already exists in the destination directory, you will be asked whether you want to overwrite it.
