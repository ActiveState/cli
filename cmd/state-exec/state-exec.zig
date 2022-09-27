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
const time = std.time;

const executorName = "state-exec";
const executorXName = "state-execx";
const envVarKeyVerbose = "ACTIVESTATE_VERBOSE";

const initMsgDataErrPrefix = "InitMsgData_";

const Error = error{
    InspectSelfPath,
    ProcessArgs,
    SetRuntimeCmd,
    SetRuntimeUserArgs,
    ChildProcInit,
    ChildProcSpawn,
};

const DebugPrint = struct {
    start: i128,
    w: fs.File.Writer,
    verbose: bool,

    const Self = @This();

    pub fn init(a: mem.Allocator, w: fs.File.Writer) DebugPrint {
        const verboseEnvVarVal = process.getEnvVarOwned(a, envVarKeyVerbose) catch "";

        return DebugPrint{
            .start = time.nanoTimestamp(),
            .w = w,
            .verbose = mem.eql(u8, verboseEnvVarVal, "true"),
        };
    }

    pub fn print(self: Self, comptime format: []const u8, args: anytype) void {
        if (!self.verbose) {
            return;
        }
        const now = time.nanoTimestamp();

        self.w.print("[{s: >12} {d: >9}] ", .{ executorName, now - self.start }) catch return;
        self.w.print(format, args) catch return;
    }
};

pub fn main() !void {
    const stderr = io.getStdErr().writer();

    const exitCode = run(stderr) catch |err| {
        try stderr.print("{s}: ", .{executorName});

        switch (err) {
            Error.InspectSelfPath => try stderr.print("Cannot inspect path of this executable.\n", .{}),
            Error.ProcessArgs => try stderr.print("Cannot process command args.\n", .{}),
            Error.SetRuntimeCmd => try stderr.print("Cannot set runtime command for child process.\n", .{}),
            Error.SetRuntimeUserArgs => try stderr.print("Cannot set user args for child process.\n", .{}),
            Error.ChildProcInit => try stderr.print("Cannot initialize child process for runtime.\n", .{}),
            Error.ChildProcSpawn => try stderr.print("Cannot spawn child process for runtime.\n", .{}),
        }

        try stderr.print("{s}: This application is not intended to be user serviceable; Please contact support for assistance.\n", .{executorName});

        process.exit(1);
    };
    os.exit(exitCode);
}

fn run(stderr: fs.File.Writer) Error!u8 {
    var arena = heap.ArenaAllocator.init(heap.page_allocator);
    defer arena.deinit();
    const a = arena.allocator();

    const debug = DebugPrint.init(a, stderr);
    debug.print("run hello\n", .{});
    defer debug.print("run goodbye\n", .{});

    const buf = a.alloc(u8, 16) catch return Error.SetRuntimeCmd; // fix error name/msg
    const pid = fmt.bufPrint(buf, "{d}", .{Thread.getCurrentId()}) catch return Error.SetRuntimeCmd; // fix error name/msg
    const execPath = fs.selfExePathAlloc(a) catch return Error.InspectSelfPath;
    debug.print("message data - pid: {s}, exec: {s}\n", .{ pid, execPath });

    var usrArgs = process.argsAlloc(a) catch return Error.ProcessArgs;
    defer process.argsFree(a, usrArgs);

    var cmdArgs = ArrayList([]const u8).init(a);
    defer cmdArgs.deinit();
    cmdArgs.append(executorXName) catch return Error.SetRuntimeCmd; // fix error name/msg
    cmdArgs.append(execPath) catch return Error.SetRuntimeCmd; // fix error name/msg
    cmdArgs.append(pid) catch return Error.SetRuntimeCmd; // fix error name/msg
    cmdArgs.appendSlice(usrArgs[1..]) catch return Error.SetRuntimeUserArgs;

    const childProc = ChildProcess.init(cmdArgs.items, a) catch return Error.ChildProcInit;
    defer childProc.deinit();

    var term = childProc.spawnAndWait() catch return Error.ChildProcSpawn;
    return term.Exited;
}
