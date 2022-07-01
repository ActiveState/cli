const std = @import("std");
const builtin = @import("builtin");
const ArrayList = std.ArrayList;
const ChildProcess = std.ChildProcess;
const fmt = std.fmt;
const fs = std.fs;
const heap = std.heap;
const io = std.io;
const mem = std.mem;
const net = std.net;
const os = std.os;
const process = std.process;
const Thread = std.Thread;

const ArgError = error{
    InvalidArgOne,
    InvalidArgTwo,
};

const RunError = process.ArgIterator.InitError || ArgError;

pub fn main() !void {
    const stderr = io.getStdErr().writer();

    run() catch |err| {
        switch (err) {
            RunError.InitError => try stderr.print("oops", .{}),
            ArgError.InvalidArgOne => try stderr.print("first arg should be a socket file\n", .{}),
            ArgError.InvalidArgTwo => try stderr.print("second arg should be a language runtime\n", .{}),
        }
        process.exit(1);
    };
}

fn sendMsgToServer(a: mem.Allocator, path: []const u8, pid: i32, exec: []const u8) !void {
    const clientMsgFmt = "heart<{d}<{s}";

    const conn = try net.connectUnixSocket(path);
    defer conn.close();

    var clientMsg = try fmt.allocPrint(a, clientMsgFmt, .{ pid, exec });
    _ = try conn.write(clientMsg);

    var buf: [1024]u8 = undefined;
    _ = try conn.read(buf[0..]);
}

pub fn run() RunError!void {
    var arena = heap.ArenaAllocator.init(heap.page_allocator);
    defer arena.deinit();
    const a = arena.allocator();

    var argIt = try process.argsWithAllocator(a);
    defer argIt.deinit();

    _ = argIt.skip();
    const path = try argIt.next(a) orelse {
        return ArgError.InvalidArgOne;
    };

    const runt = try argIt.next(a) orelse {
        return ArgError.InvalidArgTwo;
    };

    var pid: i32 = @truncate(i32, @bitCast(i64, Thread.getCurrentId()));

    const exec = try fs.selfExePathAlloc(a);

    const clientThread = try Thread.spawn(.{}, sendMsgToServer, .{ a, path, pid, exec });
    clientThread.join();

    var usrArgs = try process.argsAlloc(a);
    defer process.argsFree(a, usrArgs);

    var cmdArgs = ArrayList([]const u8).init(a);
    defer cmdArgs.deinit();
    try cmdArgs.append(runt);
    try cmdArgs.appendSlice(usrArgs[3..]);

    const childProc = try ChildProcess.init(cmdArgs.items, a);
    defer childProc.deinit();
    var term = try childProc.spawnAndWait();
    os.exit(term.Exited);
}
