// from https://zig-by-example.github.io/tcp-connection.html
const std = @import("std");
const net = std.net;
const testing = std.testing;

const clientMsg = "http-addr";

fn sendMsgToServer(path: []const u8) !void {
    const stdout = std.io.getStdOut().writer();

    const conn = net.connectUnixSocket(path) catch |err| {
        try stdout.print("{s}\n", .{err});
        return err;
    };

    defer conn.close();

    _ = try conn.write(clientMsg);

    var buf: [1024]u8 = undefined;
    var respSize = try conn.read(buf[0..]);

    _ = respSize;
    //try stdout.print("{s}\n", .{buf[0..respSize]});
}

pub fn main() !void {
    const stdout = std.io.getStdOut().writer();
    const process = std.process;

    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();

    var a: std.mem.Allocator = arena.allocator();

    var argIt = try process.argsWithAllocator(a);
    defer argIt.deinit();

    _ = argIt.skip();

    const path = try argIt.next(a) orelse {
        try stdout.print("expected first arg to be server addr\n", .{});
        return error.InvalidArgs;
    };

    const clientThread = std.Thread.spawn(.{}, sendMsgToServer, .{path}) catch |err| {
        try stdout.print("test\n", .{});
        try stdout.print("{s}\n", .{err});
        return err;
    };
    clientThread.join();

    var usrArgs = try process.argsAlloc(a);
    defer process.argsFree(a, usrArgs);

    var cmdArgs = std.ArrayList([]const u8).init(a);
    defer cmdArgs.deinit();
    try cmdArgs.append("/usr/bin/python3");
    try cmdArgs.appendSlice(usrArgs[2..]);

    const childProc = try std.ChildProcess.init(cmdArgs.items, a);
    defer childProc.deinit();
    var term = try childProc.spawnAndWait();
    std.os.exit(term.Exited);
}
