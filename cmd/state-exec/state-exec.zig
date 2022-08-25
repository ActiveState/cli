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
    InitMsgData,
    DirOfSelfPath,
    InitMetaData,
    ThreadSpawn,
    FormRuntimePath,
    ProcessArgs,
    SetRuntimeCmd,
    SetRuntimeUserArgs,
    ChildProcInit,
    ChildProcSpawn,
};

pub fn main() !void {
    const stderr = io.getStdErr().writer();

    const exitCode = run(stderr) catch |err| {
        try stderr.print("{s}: ", .{execName});

        switch (err) {
            Error.InitMsgData => try stderr.print("Cannot initialize MsgData type.\n.", .{}),
            Error.DirOfSelfPath => try stderr.print("Cannot get directory of path to this executable.\n", .{}),
            Error.InitMetaData => try stderr.print("Cannot initialize MetaData type.\n", .{}),
            Error.ThreadSpawn => try stderr.print("Cannot spawn thread for heartbeat.\n", .{}),
            Error.FormRuntimePath => try stderr.print("Cannot form runtime path.\n", .{}),
            Error.ProcessArgs => try stderr.print("Cannot process command args.\n", .{}),
            Error.SetRuntimeCmd => try stderr.print("Cannot set runtime command for child process.\n", .{}),
            Error.SetRuntimeUserArgs => try stderr.print("Cannot set user args for child process.\n", .{}),
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

    sock: []const u8,
    bin: []const u8,
    env: std.BufMap,

    pub fn init(a: mem.Allocator, execDir: []const u8) !MetaData {
        var sock: []const u8 = undefined;
        var bin: []const u8 = undefined;
        var env = std.BufMap.init(a);

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
                    const dim = try a.alloc(u8, trimmedLine.len);
                    mem.copy(u8, dim, trimmedLine);
                    sock = dim;
                },
                1 => {
                    var trimmedLine = mem.trimLeft(u8, line, MetaData.binDelim);
                    const dim = try a.alloc(u8, trimmedLine.len);
                    mem.copy(u8, dim, trimmedLine);
                    bin = dim;
                },
                2 => {
                    const trimmedLine = mem.trimLeft(u8, line, MetaData.envDelim);
                    var split = mem.split(u8, trimmedLine, MetaData.envDelim);
                    while (split.next()) |kv| {
                        const delim = '=';
                        const k = mem.sliceTo(kv, delim);
                        const v = kv[k.len + 1 ..];
                        try env.put(k, v);
                    }
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
            .env = env,
        };
    }

    pub fn deinit(self: *MetaData) void {
        self.env.deinit();
    }
};

const MsgData = struct {
    pub const fmt = "heart<{d}<{s}";

    pid: i32,
    exec: []const u8,

    pub fn init(a: mem.Allocator) !MsgData {
        return MsgData{
            .pid = @truncate(i32, @bitCast(i64, Thread.getCurrentId())),
            .exec = try fs.selfExePathAlloc(a),
        };
    }
};

fn sendMsgToServer(a: mem.Allocator, stderr: fs.File.Writer, sock: []const u8, d: MsgData) !void {
    const conn = net.connectUnixSocket(sock) catch |err| {
        try stderr.print("{s}: Cannot connect to socket: {s}.\n", .{ execName, err });
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

    const msgData = MsgData.init(a) catch return Error.InitMsgData;
    const execDir = path.dirname(msgData.exec) orelse return Error.DirOfSelfPath;
    var metaData = MetaData.init(a, execDir) catch return Error.InitMetaData;
    defer metaData.deinit();

    const clientThread = Thread.spawn(.{}, sendMsgToServer, .{ a, stderr, metaData.sock, msgData }) catch {
        return Error.ThreadSpawn;
    };
    defer clientThread.join();

    const runt = path.join(a, &[_][]const u8{ metaData.bin, path.basename(msgData.exec) }) catch return Error.FormRuntimePath;

    var usrArgs = process.argsAlloc(a) catch return Error.ProcessArgs;
    defer process.argsFree(a, usrArgs);

    var cmdArgs = ArrayList([]const u8).init(a);
    defer cmdArgs.deinit();
    cmdArgs.append(runt) catch return Error.SetRuntimeCmd;
    cmdArgs.appendSlice(usrArgs[1..]) catch return Error.SetRuntimeUserArgs;

    const childProc = ChildProcess.init(cmdArgs.items, a) catch return Error.ChildProcInit;
    defer childProc.deinit();
    childProc.env_map = &metaData.env;
    var term = childProc.spawnAndWait() catch return Error.ChildProcSpawn;
    return term.Exited;
}
