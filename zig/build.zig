// dev phase 5: build for the `lure` TFTP recovery helper.
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
            .link_libc = true, // lure calls libc sockets directly (std.net gone in 0.16)
        }),
    });
    b.installArtifact(exe);

    // `zig build test` — unit tests for the pure helpers (be16, parseRRQ).
    const unit_tests = b.addTest(.{
        .root_module = b.createModule(.{
            .root_source_file = b.path("lure.zig"),
            .target = target,
            .optimize = optimize,
            .link_libc = true,
        }),
    });
    const run_unit_tests = b.addRunArtifact(unit_tests);
    const test_step = b.step("test", "run lure unit tests");
    test_step.dependOn(&run_unit_tests.step);
}
