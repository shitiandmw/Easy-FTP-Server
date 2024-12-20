// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT
import {main} from '../models';

export function CheckAutoStart():Promise<boolean>;

export function GetDefaultConfig():Promise<main.DefaultConfig>;

export function GetServerIP():Promise<string>;

export function Greet(arg1:string):Promise<string>;

export function IsServerRunning():Promise<boolean>;

export function LoadConfig():Promise<main.DefaultConfig>;

export function MinimizeToTray():Promise<void>;

export function OpenDirectoryDialog():Promise<string>;

export function SaveConfig(arg1:main.FTPConfig):Promise<void>;

export function SetAutoStart(arg1:boolean):Promise<void>;

export function StartFTPServer(arg1:main.FTPConfig):Promise<string>;

export function StopFTPServer():Promise<string>;
