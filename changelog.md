# Changelog

## 1.0.22

- Changed timeout to only wait for the response headers
  - Download can take as long as it needs

## 1.0.12
- Show install path on missing folder question
- Add useragent to file downloads

## 1.0.11
- Updated packages
  - Fixes issue with showing error/debug messages while the progress bar is being shown
- Updated several error messages so they are clearer.
- Fixed some logs not being saved to the install log

## 1.0.10
- Changed some error messages from fatals to errors
- Changed description of some flags
- Fixed potential issue in downloading files where an error would return and not continue a for loop

## 1.0.9
- Exclude installer and installer log from empty dir checks