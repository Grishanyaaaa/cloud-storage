// Generates src/routeTree.gen.ts from the TanStack Router file-based routes
// in src/routes. Used as a prebuild step so that `tsc -b` (which runs before
// `vite build` in npm scripts) can resolve the @/routeTree.gen import.
//
// The same generation also happens during `vite build` via the
// TanStackRouterVite plugin, but we cannot rely on that here because tsc runs
// first. Keep the routesDirectory / generatedRouteTree paths in sync with
// vite.config.ts.
import { Generator, getConfig } from "@tanstack/router-generator";
import path from "node:path";

const config = await getConfig({
  routesDirectory: path.resolve("./src/routes"),
  generatedRouteTree: path.resolve("./src/routeTree.gen.ts"),
});

const generator = new Generator({ config, root: process.cwd() });
await generator.run();
