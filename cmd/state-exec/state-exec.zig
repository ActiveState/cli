const std = @import("std");
const builtin = @import("builtin");
const ArrayList = std.ArrayList;
const BufMap = std.BufMap;
const ChildProcess = std.ChildProcess;
const Thread = std.Thread;
const fmt = std.fmt;
const fs = std.fs;
const heap = std.heap;
const io = std.io;
const mem = std.mem;
const net = std.net;
const os = std.os;
const path = std.fs.path;
const process = std.process;

const execName = "state-exec";

const initMsgDataErrPrefix = "InitMsgData_";
const initMetaDataErrPrefix = "InitMetaData_";

const Error = error{
    InitMsgData_InspectSelfPath,
    DirOfSelfPath,
    InitMetaData_FormMetaFilePath,
    InitMetaData_OpenMetaFile,
    InitMetaData_ReadMetaFile,
    InitMetaData_AllocLine,
    InitMetaData_AddToMap,
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

        const errName = @errorName(err);
        if (errName.len >= initMsgDataErrPrefix.len and mem.eql(u8, errName[0..initMsgDataErrPrefix.len], initMsgDataErrPrefix)) {
            try stderr.print("Cannot initialize MsgData type: ", .{});
        }
        if (errName.len >= initMetaDataErrPrefix.len and mem.eql(u8, errName[0..initMetaDataErrPrefix.len], initMetaDataErrPrefix)) {
            try stderr.print("Cannot initialize MetaData type: ", .{});
        }

        switch (err) {
            Error.InitMsgData_InspectSelfPath => try stderr.print("Cannot inspect path of this executable.\n", .{}),
            Error.DirOfSelfPath => try stderr.print("Cannot get directory of path to this executable.\n", .{}),
            Error.InitMetaData_FormMetaFilePath => try stderr.print("Cannot form meta file path.\n", .{}),
            Error.InitMetaData_OpenMetaFile => try stderr.print("Cannot open meta file.\n", .{}),
            Error.InitMetaData_ReadMetaFile => try stderr.print("Cannot read meta file.\n", .{}),
            Error.InitMetaData_AllocLine => try stderr.print("Cannot allocate memory for line.\n", .{}),
            Error.InitMetaData_AddToMap => try stderr.print("Cannot add value to map.\n", .{}),
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

fn run(stderr: fs.File.Writer) Error!u8 {
    var arena = heap.ArenaAllocator.init(heap.page_allocator);
    defer arena.deinit();
    const a = arena.allocator();

    const msgData = try MsgData.init(a);
    const execDir = path.dirname(msgData.exec) orelse return Error.DirOfSelfPath;
    var metaData = try MetaData.init(a, execDir);
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

const MsgData = struct {
    pub const fmt = "heart<{d}<{s}";

    pid: i32,
    exec: []const u8,

    pub fn init(a: mem.Allocator) Error!MsgData {
        return MsgData{
            .pid = @truncate(i32, @bitCast(i64, Thread.getCurrentId())),
            .exec = fs.selfExePathAlloc(a) catch return Error.InitMsgData_InspectSelfPath,
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

const MetaData = struct {
    pub const filename = "meta.as";
    pub const sockDelim = "::sock::";
    pub const binDelim = "::bin::";
    pub const envDelim = "::env::";
    pub const envVarDelim = '=';

    sock: []const u8,
    bin: []const u8,
    env: BufMap,

    pub fn init(a: mem.Allocator, execDir: []const u8) Error!MetaData {
        var sock: []const u8 = undefined;
        var bin: []const u8 = undefined;

        var env = BufMap.init(a);
        for (os.environ) |envEntry| {
            const k = mem.sliceTo(envEntry, envVarDelim);
            const v = envEntry[k.len + 1 .. mem.len(envEntry)];
            env.put(k, v) catch return Error.InitMetaData_AddToMap;
        }

        const metaPath = path.join(a, &[_][]const u8{ execDir, MetaData.filename }) catch return Error.InitMetaData_FormMetaFilePath;
        const metaFile = fs.openFileAbsolute(metaPath, .{ .read = true }) catch return Error.InitMetaData_OpenMetaFile;
        defer metaFile.close();

        const metaReader = metaFile.reader();
        var metaBuf: [32760]u8 = undefined;
        var lineCt: i32 = 0;
        while (metaReader.readUntilDelimiterOrEof(&metaBuf, '\n') catch return Error.InitMetaData_ReadMetaFile) |line| : (lineCt += 1) {
            switch (lineCt) {
                0 => {
                    const trimmedLine = mem.trimLeft(u8, line, MetaData.sockDelim);
                    const dim = a.alloc(u8, trimmedLine.len) catch return Error.InitMetaData_AllocLine;
                    mem.copy(u8, dim, trimmedLine);
                    sock = dim;
                },
                1 => {
                    var trimmedLine = mem.trimLeft(u8, line, MetaData.binDelim);
                    const dim = a.alloc(u8, trimmedLine.len) catch return Error.InitMetaData_AllocLine;
                    mem.copy(u8, dim, trimmedLine);
                    bin = dim;
                },
                2 => {
                    const trimmedLine = mem.trimLeft(u8, line, MetaData.envDelim);
                    var split = mem.split(u8, trimmedLine, MetaData.envDelim);
                    while (split.next()) |kv| {
                        const k = mem.sliceTo(kv, envVarDelim);
                        const v = kv[k.len + 1 ..];
                        env.put(k, v) catch return Error.InitMetaData_AddToMap;
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
