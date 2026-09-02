// PLANNED (dev phase 5). Placeholder build for the `lure` recovery helper.
// Zig 0.16 API (addExecutable takes a root_module).
const std = @import("std");

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});
    const exe = b.addExecutable(.{
        .name = "lure",
        .root_module = b.createModule(.{
            .root_source_file = b.path("lure.zig"),
            .target = target,
            .optimize = optimize,
        }),
    });
    b.installArtifact(exe);
}
