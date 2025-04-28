# FTB Server Installer

Todo: Write a project description

## Usage

This usage guide assumes that the server installed is named `serverinstaller.exe`, the installer downloaded may have a different name such as `ftb-server-windows-amd64.exe` or `serverinstaller_<pack_id>_<version_id>.exe`.

### Windows

You can either double-click on the installer to run it, or you can run it from the command line.

To run from the command line, open a command prompt and navigate to the directory where the installer is located. You can then run the installer with the following command:

```cmd
.\serverinstaller.exe -pack <pack_id> -version <version_id>
```

### MacOS/Linux

Open up a terminal and navigate to the directory where the installer is located. You can then run the installer with the following command:

```cmd
./serverinstaller -pack <pack_id> -version <version_id>
```

### Flags

| Flag              | Default              | Description                                                                                                         |
|-------------------|----------------------|---------------------------------------------------------------------------------------------------------------------|
| `-dir`            | `./`                 | Directory to install the server files in (Defaults to current directory)                                            |
| `-auto`           | `false`              | Doesn't ask questions, just runs the installer                                                                      |
| `-pack`           |                      | The ID of the modpack you would like to install                                                                     |
| `-version`        |                      | ID of the modpack version you would like to install, if not set, latest stable release will be selected             |
| `-latest`         | `false`              | If the version id is not set, and this flag is used, it will get the latest stable, beta or alpha version available |
| `-validate`       | `false`              | Validates the modpack files after they have been downloaded and installed                                           |
| `-provider`       | `ftb`                | Sets the modpack provider (ftb is the only provider at the moment)                                                  |
| `-force`          |                      | Only works when -auto is used, will force the installer to continue upon warnings                                   |
| `-threads`        | 4                    | Number of concurrent download threads                                                                               |
| `-apikey`         |                      | API key for accessing private modpacks                                                                              |
| `-skip-modloader` | `false`              | If set, installer will skip running the modloader installer                                                         |
| `-no-java`        | `false`              | If set, installer wont download a copy of java                                                                      |
| `-no-colours`     | `false`              | Removes the colour formatting from the console output                                                               |
| `-verbose`        | `false`              | Enables debug logging                                                                                               |

## Looking for a Modded Minecraft Server? `Ad`

[![Promotion](https://cdn.feed-the-beast.com/assets/promo/ftb-bh-promo-large.png)](https://bisecthosting.com/ftb)
