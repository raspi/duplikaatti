# duplikaatti

![GitHub All Releases](https://img.shields.io/github/downloads/raspi/duplikaatti/total?style=for-the-badge)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/raspi/duplikaatti?style=for-the-badge)
![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/raspi/duplikaatti?style=for-the-badge)


Remove duplicate files and do it fast. `duplikaatti` is designed to go through 50 TiB+ of data and hundreds of thousands of files and find duplicate files in few minutes.

## Algorithm
* Create file list of given directories 
  * do not add files with same identifier already added to the list (windows: file id, *nix: inode)
  * do not add 0 byte files
  * directories listed first has higher priority than the last
* Remove all files which do not share same file sizes (ie. there's only one 1000 byte file -> remove)
* Read first bytes of files and generate SHA256 sum of those bytes
* Remove all hashes which occured only once
* Read last bytes of files and generate SHA256 sum of those bytes
* Remove all hashes which occured only once
* Now finally hash the whole files that are left
* Remove all hashes which occured only once
* Generate list of files to keep and what to remove
  * use directory priority and file age to find what to keep 
    * oldest and highest priority files are kept
* Finally, remove files

## Usage
```
Duplicate file remover (version 1.0.0)
Removes duplicate files. Algorithm idea from rdfind.

Usage of duplikaatti [options] <directories>:

Parameters:
  -remove
    	Actually remove files.

Examples:
  Test what would be removed:
    duplikaatti /home/raspi/storage /mnt/storage
  Remove files:
    duplikaatti -remove /home/raspi/storage /mnt/storage
```

Idea inspired by https://github.com/pauldreik/rdfind
