// from https://zig-by-example.github.io/tcp-connection.html
const std = @import("std");
const builtin = @import("builtin");
const net = std.net;
const testing = std.testing;

const clientMsgFmt = "heart:{d}";

fn sendMsgToServer(a: std.mem.Allocator, path: []const u8, pid: i32) !void {
    const conn = try net.connectUnixSocket(path);
    defer conn.close();

    var clientMsg = try std.fmt.allocPrint(a, clientMsgFmt, .{pid});
    _ = try conn.write(clientMsg);

    var buf: [1024]u8 = undefined;
    _ = try conn.read(buf[0..]);
}

pub fn main() !void {
    const stderr = std.io.getStdErr().writer();
    const process = std.process;

    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();

    var a: std.mem.Allocator = arena.allocator();

    var argIt = try process.argsWithAllocator(a);
    defer argIt.deinit();

    _ = argIt.skip();

    const path = try argIt.next(a) orelse {
        try stderr.print("first arg should be a path to socket file\n", .{});
        return error.InvalidArgs;
    };

    const runt = try argIt.next(a) orelse {
        try stderr.print("second arg should be a path to a language runtime", .{});
        return error.InvalidArgs;
    };

    var pid: i32 = undefined;
    switch (builtin.os.tag) {
            .windows => {
                pid = @bitCast(i32,std.os.windows.kernel32.GetCurrentProcessId());
            },
            else => {
                pid = std.os.system.getpid();
            },
    }

    const clientThread = try std.Thread.spawn(.{}, sendMsgToServer, .{a, path, pid});
    clientThread.join();

    var usrArgs = try process.argsAlloc(a);
    defer process.argsFree(a, usrArgs);

    var cmdArgs = std.ArrayList([]const u8).init(a);
    defer cmdArgs.deinit();
    try cmdArgs.append(runt);
    try cmdArgs.appendSlice(usrArgs[3..]);

    const childProc = try std.ChildProcess.init(cmdArgs.items, a);
    defer childProc.deinit();
    var term = try childProc.spawnAndWait();
    std.os.exit(term.Exited);
}
