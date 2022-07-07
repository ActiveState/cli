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

const Error = error{
    ArgIterator,
    ArgInvalidOne,
    ArgMissingOne,
    ArgInvalidTwo,
    ArgMissingTwo,
    ArgCollector,
    ArgCollectRunt,
    ArgCollectUsr,
    InspectSelfPath,
    ThreadSpawn,
    ChildProcInit,
    ChildProcSpawn,
};

pub fn main() !void {
    const stderr = io.getStdErr().writer();

    run() catch |err| {
        switch (err) {
            Error.ArgIterator => try stderr.print("cannot process args", .{}),
            Error.ArgInvalidOne, Error.ArgMissingOne => try stderr.print("first arg should be a socket file\n", .{}),
            Error.ArgInvalidTwo, Error.ArgMissingTwo => try stderr.print("second arg should be a language runtime\n", .{}),
            Error.ArgCollector => try stderr.print("cannot setup arg collector", .{}),
            Error.ArgCollectRunt => try stderr.print("cannot collect runtime arg", .{}),
            Error.ArgCollectUsr => try stderr.print("cannot collect user args", .{}),
            Error.InspectSelfPath => try stderr.print("cannot obtain path to this executable", .{}),
            Error.ThreadSpawn => try stderr.print("cannot spawn thread for heartbeat", .{}),
            Error.ChildProcInit => try stderr.print("cannot initialize child process for runtime", .{}),
            Error.ChildProcSpawn => try stderr.print("cannot spawn child process for runtime", .{}),
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

pub fn run() Error!void {
    var arena = heap.ArenaAllocator.init(heap.page_allocator);
    defer arena.deinit();
    const a = arena.allocator();

    var argIt = process.argsWithAllocator(a) catch return Error.ArgIterator;
    defer argIt.deinit();

    _ = argIt.skip();
    const path = (argIt.next(a) orelse return Error.ArgMissingOne) catch return Error.ArgInvalidOne;
    const runt = (argIt.next(a) orelse return Error.ArgMissingTwo) catch return Error.ArgInvalidTwo;

    var pid: i32 = @truncate(i32, @bitCast(i64, Thread.getCurrentId()));

    const exec = fs.selfExePathAlloc(a) catch return Error.InspectSelfPath;

    const clientThread = Thread.spawn(.{}, sendMsgToServer, .{ a, path, pid, exec }) catch {
        return Error.ThreadSpawn;
    };
    clientThread.join();

    var usrArgs = process.argsAlloc(a) catch return Error.ArgCollector;
    defer process.argsFree(a, usrArgs);

    var cmdArgs = ArrayList([]const u8).init(a);
    defer cmdArgs.deinit();
    cmdArgs.append(runt) catch return Error.ArgCollectRunt;
    cmdArgs.appendSlice(usrArgs[3..]) catch return Error.ArgCollectUsr;

    const childProc = ChildProcess.init(cmdArgs.items, a) catch return Error.ChildProcInit;
    defer childProc.deinit();
    var term = childProc.spawnAndWait() catch return Error.ChildProcSpawn;
    os.exit(term.Exited);
}
