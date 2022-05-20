// from https://zig-by-example.github.io/tcp-connection.html
const std = @import("std");
const net = std.net;
const testing = std.testing;

const client_msg = "http-addr";

fn sendMsgToServer(path: []const u8) !void {
    const stdout = std.io.getStdOut().writer();

    const conn = net.connectUnixSocket(path) catch |err| {
        try stdout.print("{s}\n", .{err});
        return err;
    };

    defer conn.close();

    _ = try conn.write(client_msg);

    var buf: [1024]u8 = undefined;
    var resp_size = try conn.read(buf[0..]);

    _ = resp_size;
    //try stdout.print("{s}\n", .{buf[0..resp_size]});
}

pub fn main() !void {
    const stdout = std.io.getStdOut().writer();
    const process = std.process;

    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();

    var a: std.mem.Allocator = arena.allocator();

    var arg_it = try process.argsWithAllocator(a);
    _ = arg_it.skip();

    const path = try arg_it.next(a) orelse {
        try stdout.print("expected first arg to be server addr\n", .{});
        return error.InvalidArgs;
    };

    const client_thread = std.Thread.spawn(.{}, sendMsgToServer, .{path}) catch |err| {
        try stdout.print("test\n", .{});
        try stdout.print("{s}\n", .{err});
        return err;
    };
    defer client_thread.join();
}
