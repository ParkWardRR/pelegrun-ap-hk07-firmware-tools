//! lure — a minimal read-only TFTP (RFC 1350) responder for u-boot deep-brick
//! recovery (`tftpboot`). Serves files from a root directory over UDP.
//!
//! Serves the current directory on :6969 (recovery uses :69, which needs root).
//! Unofficial; part of pelegrun-ap-hk07-firmware-tools.
//!
//! Zig 0.16 removed std.net and the std.posix socket wrappers (networking is
//! moving into std.Io), so lure talks to libc sockets directly — the stable,
//! portable surface. std.c still provides the OS-correct sockaddr/const layouts.
const std = @import("std");
const c = std.c;

const OP_RRQ: u16 = 1;
const OP_DATA: u16 = 3;
const OP_ACK: u16 = 4;
const OP_ERROR: u16 = 5;
const BLOCK: usize = 512;
const PORT: u16 = 6969; // recovery uses :69 (needs root)

// libc socket surface (declared here; std.posix no longer wraps these in 0.16).
extern "c" fn socket(domain: c_int, sock_type: c_int, protocol: c_int) c_int;
extern "c" fn close(fd: c_int) c_int;
extern "c" fn bind(fd: c_int, addr: *const anyopaque, len: c.socklen_t) c_int;
extern "c" fn setsockopt(fd: c_int, level: c_int, optname: c_int, optval: *const anyopaque, len: c.socklen_t) c_int;
extern "c" fn recvfrom(fd: c_int, buf: [*]u8, len: usize, flags: c_int, src: ?*anyopaque, srclen: ?*c.socklen_t) isize;
extern "c" fn sendto(fd: c_int, buf: [*]const u8, len: usize, flags: c_int, dst: ?*const anyopaque, dstlen: c.socklen_t) isize;
// libc file surface (std.fs moved into std.Io.Dir in 0.16); O_RDONLY == 0.
extern "c" fn open(path: [*:0]const u8, flags: c_int) c_int;
extern "c" fn read(fd: c_int, buf: [*]u8, count: usize) isize;
const O_RDONLY: c_int = 0;

fn be16(hi: u8, lo: u8) u16 {
    return (@as(u16, hi) << 8) | @as(u16, lo);
}

/// Parse a TFTP packet as a read request (RRQ): returns the requested filename,
/// or null if it isn't a well-formed RRQ (wrong opcode, runt, or no NUL).
fn parseRRQ(pkt: []const u8) ?[]const u8 {
    if (pkt.len < 4) return null;
    if (be16(pkt[0], pkt[1]) != OP_RRQ) return null;
    const rest = pkt[2..];
    const nul = std.mem.indexOfScalar(u8, rest, 0) orelse return null;
    return rest[0..nul];
}

pub fn main() !void {
    // v1 config: serve the current directory on PORT. A future revision reads
    // these from pelegrun once the 0.16 env/args API settles.
    const port: u16 = PORT;
    const root: []const u8 = ".";

    const fd = socket(c.AF.INET, c.SOCK.DGRAM, 0);
    if (fd < 0) return error.SocketFailed;
    defer _ = close(fd);

    var addr = c.sockaddr.in{
        .port = std.mem.nativeToBig(u16, port),
        .addr = 0, // 0.0.0.0
    };
    if (bind(fd, &addr, @sizeOf(c.sockaddr.in)) != 0) return error.BindFailed;

    // 3s recv timeout so a stalled client can't hang us forever
    const tv = c.timeval{ .sec = 3, .usec = 0 };
    _ = setsockopt(fd, c.SOL.SOCKET, c.SO.RCVTIMEO, &tv, @sizeOf(c.timeval));

    std.debug.print("lure: TFTP recovery responder on :{d} serving {s}\n", .{ port, root });

    var pkt: [BLOCK + 4]u8 = undefined;
    while (true) {
        var src: c.sockaddr.in = undefined;
        var slen: c.socklen_t = @sizeOf(c.sockaddr.in);
        const n = recvfrom(fd, &pkt, pkt.len, 0, &src, &slen);
        if (n < 4) continue; // timeout (-1) or runt packet — keep listening
        const len: usize = @intCast(n);
        const name = parseRRQ(pkt[0..len]) orelse continue;
        serveFile(fd, root, name, &src, slen) catch |e| {
            sendError(fd, &src, slen, 1, "file not found");
            std.debug.print("lure: {s}: {any}\n", .{ name, e });
        };
    }
}

fn serveFile(fd: c_int, root: []const u8, name: []const u8, dst: *const c.sockaddr.in, dlen: c.socklen_t) !void {
    var pathbuf: [1024]u8 = undefined;
    // reject path traversal outright — a recovery client only fetches by name
    if (std.mem.indexOf(u8, name, "..") != null) return error.BadName;
    const path = try std.fmt.bufPrintZ(&pathbuf, "{s}/{s}", .{ root, name });
    const ffd = open(path.ptr, O_RDONLY);
    if (ffd < 0) return error.FileNotFound;
    defer _ = close(ffd);

    var data: [BLOCK + 4]u8 = undefined;
    var ack: [4]u8 = undefined;
    var block: u16 = 1;
    while (true) {
        const body = data[4 .. 4 + BLOCK];
        const rn = read(ffd, body.ptr, BLOCK);
        if (rn < 0) return error.ReadFailed;
        const got: usize = @intCast(rn);
        data[0] = 0;
        data[1] = OP_DATA;
        data[2] = @intCast((block >> 8) & 0xff);
        data[3] = @intCast(block & 0xff);
        if (sendto(fd, &data, 4 + got, 0, dst, dlen) < 0) return error.SendFailed;

        // wait for the matching ACK (recv timeout bounds the wait)
        const an = recvfrom(fd, &ack, ack.len, 0, null, null);
        if (an >= 4 and be16(ack[0], ack[1]) == OP_ACK and be16(ack[2], ack[3]) == block) {
            if (got < BLOCK) return; // final (short) block acked — done
            block +%= 1;
        } else if (an < 0) {
            return error.AckTimeout;
        }
    }
}

fn sendError(fd: c_int, dst: *const c.sockaddr.in, dlen: c.socklen_t, code: u16, msg: []const u8) void {
    var e: [128]u8 = undefined;
    e[0] = 0;
    e[1] = OP_ERROR;
    e[2] = @intCast((code >> 8) & 0xff);
    e[3] = @intCast(code & 0xff);
    const m = @min(msg.len, e.len - 5);
    @memcpy(e[4 .. 4 + m], msg[0..m]);
    e[4 + m] = 0;
    _ = sendto(fd, &e, 5 + m, 0, dst, dlen);
}

// ---- unit tests (`zig build test`) ----

test "be16 combines bytes big-endian" {
    try std.testing.expectEqual(@as(u16, 0x0103), be16(0x01, 0x03));
    try std.testing.expectEqual(@as(u16, OP_RRQ), be16(0, 1));
    try std.testing.expectEqual(@as(u16, 0xFFFF), be16(0xFF, 0xFF));
}

test "parseRRQ extracts the filename" {
    const pkt = [_]u8{ 0, OP_RRQ } ++ "fw.bin".* ++ [_]u8{0} ++ "octet".* ++ [_]u8{0};
    const name = parseRRQ(pkt[0..]) orelse return error.TestExpectedName;
    try std.testing.expectEqualStrings("fw.bin", name);
}

test "parseRRQ rejects non-RRQ, runts, and missing NUL" {
    try std.testing.expect(parseRRQ(&[_]u8{ 0, OP_ACK, 0, 1 }) == null); // wrong opcode
    try std.testing.expect(parseRRQ(&[_]u8{ 0, OP_RRQ }) == null); // too short (<4)
    try std.testing.expect(parseRRQ(&[_]u8{ 0, OP_RRQ, 0x41, 0x42 }) == null); // no NUL
}
