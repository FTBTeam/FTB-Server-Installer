# Changelog

## 1.0.41

- Fix: NeoForge run.sh/bat not being updated to use own java on 26.x
- Updated packages

## 1.0.40

- Updated to golang 1.26
- Removed logging fix (now fixed in pterm itself)
- Updated packages

## 1.0.39

- Fix for forge file name regex

## 1.0.38

- Only rename MC server jar on specific versions 

## 1.0.37

- Check env variable (FTB_MODPACK_API_KEY) for API key

## 1.0.36

- Fix: dont rename MC jar on 1.12.2

## 1.0.35

- Fix 1.5.2 start script

## 1.0.34

- Check content length on modloader downloads

## 1.0.33

- Fix: Missing return on success/failed install

## 1.0.32

- Updated packages
- Fix: `nogui` not being set
- Fix: Close response body
- Don't allow download threads to be less than 1
- Fix: potential resource leaks and nil checks

## 1.0.31

- Fix: linux terminal not opening on double click
- Updated packages

## 1.0.30

- Updated API auth headers
- Updated packages

## 1.0.29

- Fix nogui not being added if you use system java

## 1.0.28

- Fixed typo in Minecraft Forge install error message
- Updated golang to 1.25
- Updated packages

## 1.0.27

- Added new flag `-accept-eula`
  - Automatically accepts the Minecraft EULA

## 1.0.24

- Rewrite download handler
  - It will now retry a download 3 times before moving on to the next available mirror or erroring out
- Added modpack id and version id next to the info message

## 1.0.23

- Added debug print to show full error message on download failure

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