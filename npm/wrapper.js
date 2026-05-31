import { existsSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

export function toGoTarget(platform = process.platform, arch = process.arch) {
  const goosByPlatform = new Map([
    ["darwin", "darwin"],
    ["linux", "linux"],
    ["win32", "windows"],
  ]);
  const goarchByArch = new Map([
    ["arm", "arm"],
    ["arm64", "arm64"],
    ["x64", "amd64"],
  ]);
  const goos = goosByPlatform.get(platform);
  const goarch = goarchByArch.get(arch);
  if (!goos || !goarch) {
    throw new Error(`Unsupported platform: ${platform}-${arch}`);
  }
  return { goos, goarch, ext: goos === "windows" ? ".exe" : "" };
}

export function resolveBinary(rootDir = dirname(fileURLToPath(import.meta.url)), platform = process.platform, arch = process.arch) {
  const target = toGoTarget(platform, arch);
  const platformBinary = join(rootDir, `jocohunt-${target.goos}-${target.goarch}${target.ext}`);
  if (existsSync(platformBinary)) {
    return platformBinary;
  }
  const fallback = join(rootDir, `jocohunt${target.ext}`);
  if (existsSync(fallback)) {
    return fallback;
  }
  throw new Error(`No jocohunt binary found for ${process.platform}-${process.arch}`);
}
