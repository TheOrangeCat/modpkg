# modpkg
This is a pretty basic Minecraft Modpack "manager" that I cobbled together in a few hours.
It takes in a Flame-like manifest.json and produces a zip file containing a filled-in manifest.json and the overrides folder specified in the input file.
The input doesn't require fileID filled in and also has some custom variables.
## Additional variables
`modpkgver`: overrides the Minecraft mod version to search for. (string)

`modpkgIsForge`: needed for mods that don't have `Forge` specified in their version. (boolean)
