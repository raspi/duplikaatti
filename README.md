# duplikaatti
Remove duplicate files. 

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

Algorithm:
  1. Get file list from given directory list.
  2. Remove all non-unique files (files in same path).
  3. Remove all orphans (only one file with same size).
  4. Read first 1048576 bytes (1.0 MiB) of files.
  5. Remove all orphans.
  6. Read last 1048576 bytes (1.0 MiB) of files.
  7. Remove all orphans.
  8. Hash whole files.
  9. Select first file with checksum X not to be removed and add rest of the files to a duplicates list.
  10. Remove duplicates.

(c) Pekka Järvinen 2018- / https://github.com/raspi/duplikaatti
```

Idea inspired by https://github.com/pauldreik/rdfind
