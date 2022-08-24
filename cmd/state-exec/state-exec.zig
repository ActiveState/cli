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
const path = std.fs.path;
const process = std.process;
const Thread = std.Thread;

const execName = "state-exec";

const Error = error{
    InspectSelfPath,
    DirOfSelfPath,
    PathOfMeta,
    MetaOpen,
    MetaRead,
    ThreadSpawn,
    ChildProcInit,
    ChildProcSpawn,
};

pub fn main() !void {
    const stderr = io.getStdErr().writer();

    const exitCode = run(stderr) catch |err| {
        try stderr.print("{s}: ", .{execName});

        switch (err) {
            Error.InspectSelfPath => try stderr.print("Cannot obtain path to this executable.\n", .{}),
            Error.DirOfSelfPath => try stderr.print("Cannot get directory of path to this executable.\n", .{}),
            Error.PathOfMeta => try stderr.print("Cannot get path of meta data file.\n", .{}),
            Error.MetaOpen => try stderr.print("Cannot open the meta file\n", .{}),
            Error.MetaRead => try stderr.print("Cannot read the meta file\n", .{}),
            Error.ThreadSpawn => try stderr.print("Cannot spawn thread for heartbeat.\n", .{}),
            Error.ChildProcInit => try stderr.print("Cannot initialize child process for runtime.\n", .{}),
            Error.ChildProcSpawn => try stderr.print("Cannot spawn child process for runtime.\n", .{}),
        }

        try stderr.print("{s}: This application is not intended to be user serviceable; Please contact support for assistance.\n", .{execName});

        process.exit(1);
    };
    os.exit(exitCode);
}

const MetaData = struct {
    pub const filename = "meta.as";
    pub const sockDelim = "::sock::";
    pub const binDelim = "::bin::";
    pub const envDelim = "::env::";

    sock: []u8,
    bin: []u8,
    env: [][]u8,
};

fn makeMetaData(a: mem.Allocator, stderr: fs.File.Writer, execDir: []const u8) !MetaData {
    var sockBuf: [256]u8 = undefined;
    var sock: []u8 = undefined;
    var binBuf: [256]u8 = undefined;
    var bin: []u8 = undefined;

    const metaPath = try path.join(a, &[_][]const u8{ execDir, MetaData.filename });
    const metaFile = try fs.openFileAbsolute(metaPath, .{ .read = true });
    defer metaFile.close();

    const metaReader = metaFile.reader();
    var metaBuf: [32760]u8 = undefined;
    var lineCt: i32 = 0;
    while (try metaReader.readUntilDelimiterOrEof(&metaBuf, '\n')) |line| : (lineCt += 1) {
        switch (lineCt) {
            0 => {
                const trimmedLine = mem.trimLeft(u8, line, MetaData.sockDelim);
                sock = sockBuf[0..trimmedLine.len];
                mem.copy(u8, sock, trimmedLine);
                try stderr.print("case 0: {s}\n", .{sock});
            },
            1 => {
                const trimmedLine = mem.trimLeft(u8, line, MetaData.binDelim);
                bin = binBuf[0..trimmedLine.len];
                mem.copy(u8, bin, trimmedLine);
                try stderr.print("case 1: {s}\n", .{bin});
            },
            2 => {
                break;
            },
            else => {
                break;
            },
        }
    }

    return MetaData{
        .sock = sock,
        .bin = bin,
        .env = undefined,
    };
}

const MsgData = struct {
    pub const fmt = "heart<{d}<{s}";

    pid: i32,
    exec: []const u8,
};

fn makeMsgData(a: mem.Allocator) !MsgData {
    return MsgData{
        .pid = @truncate(i32, @bitCast(i64, Thread.getCurrentId())),
        .exec = try fs.selfExePathAlloc(a),
    };
}

fn sendMsgToServer(a: mem.Allocator, stderr: fs.File.Writer, sock: []const u8, d: MsgData) !void {
    const conn = net.connectUnixSocket(sock) catch |err| {
        try stderr.print("{s}: Cannot connect to socket: {s}.\n", .{ execName, err });
        try stderr.print("{s}\n", .{sock});
        return;
    };
    defer conn.close();

    var clientMsg = try fmt.allocPrint(a, MsgData.fmt, .{ d.pid, d.exec });
    _ = conn.write(clientMsg) catch |err| {
        try stderr.print("{s}: Cannot write to socket connection: {s}.\n", .{ execName, err });
        return;
    };

    var buf: [1024]u8 = undefined;
    _ = conn.read(buf[0..]) catch |err| {
        try stderr.print("{s}: Cannot read from socket connection: {s}.\n", .{ execName, err });
        return;
    };
}

fn run(stderr: fs.File.Writer) Error!u8 {
    var arena = heap.ArenaAllocator.init(heap.page_allocator);
    defer arena.deinit();
    const a = arena.allocator();

    const msgData = makeMsgData(a) catch return Error.InspectSelfPath;
    const execDir = path.dirname(msgData.exec) orelse return Error.DirOfSelfPath;
    const metaData = makeMetaData(a, stderr, execDir) catch return Error.InspectSelfPath;
    const runt = path.join(a, &[_][]const u8{ metaData.bin, path.basename(msgData.exec) }) catch return Error.InspectSelfPath;

    stderr.print("runt: {s}\n", .{runt}) catch return error.InspectSelfPath;

    const clientThread = Thread.spawn(.{}, sendMsgToServer, .{ a, stderr, metaData.sock, msgData }) catch {
        return Error.ThreadSpawn;
    };
    defer clientThread.join();

    var usrArgs = process.argsAlloc(a) catch return Error.InspectSelfPath;
    defer process.argsFree(a, usrArgs);

    var cmdArgs = ArrayList([]const u8).init(a);
    defer cmdArgs.deinit();
    cmdArgs.append(runt) catch return Error.InspectSelfPath;
    cmdArgs.appendSlice(usrArgs[6..]) catch return Error.InspectSelfPath;

    const childProc = ChildProcess.init(cmdArgs.items, a) catch return Error.ChildProcInit;
    defer childProc.deinit();
    var term = childProc.spawnAndWait() catch return Error.ChildProcSpawn;
    return term.Exited;
}
