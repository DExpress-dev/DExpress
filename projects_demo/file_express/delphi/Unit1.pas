﻿unit Unit1;

interface

uses
  Winapi.Windows, Winapi.Messages, System.SysUtils, System.Variants, System.Classes, Vcl.Graphics,
  Vcl.Controls, Vcl.Forms, Vcl.Dialogs, Vcl.Themes, Vcl.StdCtrls, Vcl.Buttons,
  Vcl.ComCtrls, Vcl.ExtCtrls, CommCtrl, System.Net.URLClient, Filectrl,
  System.Net.HttpClient, System.Net.HttpClientComponent, System.NetEncoding, System.JSON, uList,
  System.ImageList, Vcl.ImgList;

type
  PServerAddrInfo = ^TServerAddrInfo;
  TServerAddrInfo = packed record
    ServerName: string;
    Ip: string;
    Port: Integer;
    Encrypt: Boolean;
  end;

  //定义回调函数类型;
  OnLogin_CallBack = procedure(express_handle: Integer;  remote_ip: PAnsichar; remote_port: Integer; session: PAnsichar);stdcall;
  OnProgress_CallBack = function(express_handle: Integer; file_path: PAnsichar; max: Integer; cur: Integer): Boolean;stdcall;
  OnFinish_CallBack = procedure(express_handle: Integer; file_path: PAnsichar; size: Double);stdcall;
  OnDisconnect_CallBack = procedure(express_handle: Integer; remote_ip: PAnsichar; remote_port: Integer);stdcall;
  OnError_CallBack = procedure(express_handle: Integer; errorid: Integer; remote_ip: PAnsichar; remote_port: Integer);stdcall;


  {接口}
  function open_client( bindIp: PAnsichar;
                        remoteIp: PAnsichar;
                        remotePort: Integer;
                        log: PAnsichar;
                        harqSoPath: PAnsichar;
                        session: PAnsichar;
                        encrypted: Boolean;
                        onLogin: OnLogin_CallBack;
                        onProgress: OnProgress_CallBack;
                        onFinish: OnFinish_CallBack;
                        onDisconnect: OnDisconnect_CallBack;
                        onError: OnError_CallBack):Integer;stdcall;external 'client_32.dll';
  function send_file( expressHandle: Integer; filePath: PAnsichar; saveRelativePath: PAnsichar):Boolean;stdcall;external 'client_32.dll';
  function send_dir(expressHandle: Integer; dirPath: PAnsichar; saveRelativePath: PAnsichar):Boolean;stdcall;external 'client_32.dll';
  procedure close_client(expressHandle: Integer);stdcall;external 'client_32.dll';
  function version():PAnsichar ;stdcall;external 'client_32.dll';

type
  TMainFrm = class(TForm)
    Panel1: TPanel;
    Panel2: TPanel;
    PageControl1: TPageControl;
    TabSheet1: TTabSheet;
    TaskView: TListView;
    StatusBar1: TStatusBar;
    BitBtn1: TBitBtn;
    TaskTimer: TTimer;
    Panel3: TPanel;
    Panel4: TPanel;
    Panel6: TPanel;
    Panel5: TPanel;
    LocalDirEdt: TEdit;
    DirBut: TButton;
    LocalListView: TListView;
    Panel8: TPanel;
    RemoteListView: TListView;
    Panel7: TPanel;
    Label1: TLabel;
    SendBut: TButton;
    GetHttpClient: TNetHTTPClient;
    BitBtn2: TBitBtn;
    Panel9: TPanel;
    Label2: TLabel;
    ServerComboBox: TComboBox;
    Panel10: TPanel;
    DirDialog: TOpenDialog;
    IcoImageList: TImageList;
    BitBtn3: TBitBtn;
    procedure TaskViewCustomDrawItem(Sender: TCustomListView; Item: TListItem;
      State: TCustomDrawState; var DefaultDraw: Boolean);
    procedure BitBtn1Click(Sender: TObject);
    procedure FormActivate(Sender: TObject);
    procedure TaskTimerTimer(Sender: TObject);
    procedure GetHttpClientRequestCompleted(const Sender: TObject;
      const AResponse: IHTTPResponse);
    procedure BitBtn2Click(Sender: TObject);
    procedure FormCreate(Sender: TObject);
    procedure DirButClick(Sender: TObject);
    procedure BitBtn3Click(Sender: TObject);

  private
    { Private declarations }
    procedure DrawSubItem(ALV: TListView; AItem: TListItem; ASubItem: Integer; APosition: Single; AMax, AStyle: Integer; AIsShowProgress: Boolean; ADrawColor: TColor = $00005B00; AFrameColor: TColor = $00002F00);
    function ReDrawItem(AHwndLV: HWND; AItemIndex: integer): boolean;

  private
    function GetFileName(filePath: string): String;
    function GetFileSizeByName(filePath: string): Int64;

  private
    //服务器管理;
    ServerList: TDoubleList;
    function AnalysisServer(context: string):Boolean;
    procedure addServer(ServerName: string; Ip: string; Port: Integer; Encrypt: Boolean);
    function findServer(ServerName: string):PServerAddrInfo;
    procedure clearServer();
    procedure drawServer();
    procedure testServer();
  private
    //dll管理
    harqPath: string;
    clientPath: string;
    dllHandle: THandle;
    expressHandle: Integer;
    procedure openClient(bindIp, remoteIp: string; remotePort: Integer; logPath, harqPath, session: string);

  private
    procedure drawLocalDir(localDir: string);


  public
    procedure AddFile(filePath: string);
    procedure SetFileProgress(fileName: string; progress: Integer);
    function GetServer(): boolean;
  public
    { Public declarations }

  end;

var
  MainFrm: TMainFrm;

implementation

{$R *.dfm}

//回调函数定义
procedure OnExpressLogin(expressHandle: Integer; remoteIp: PAnsichar; remotePort: Integer; session: PAnsichar); stdcall;
begin
  Application.MessageBox(Pchar(remoteIp), Pchar(session), 0);
end;

procedure OnExpressFinish(expressHandle: Integer; filePath: Pchar; size: Int64); stdcall;
begin

end;

function OnExpressProgress(expressHandle: Integer; filePath: Pchar; max: Integer; cur: Integer): Boolean; stdcall;
begin

end;

procedure OnExpressDisconnect(expressHandle: Integer; remoteIp: Pchar; remotePort: Integer); stdcall;
begin

end;

procedure OnExpressError(expressHandle: Integer; errorId: Integer; remoteIp: Pchar; remotePort: Integer); stdcall;
begin

end;

procedure TMainFrm.AddFile(filePath: string);
var
  NewItem: TListItem;
  fileName: string;
  fileSize: Int64;
  nowTime: string;
begin
  fileName:= GetFileName(filePath);
  fileSize:= GetFileSizeByName(filePath);
  NewItem := TaskView.Items.Add;

  nowTime := FormatDateTime('yyyy-mm-dd hh:nn:ss', now());
  NewItem.Caption:=nowTime;

  NewItem.SubItems.Add(fileName);
  NewItem.SubItems.Add(IntToStr(fileSize));
  NewItem.SubItems.Add('0');
  NewItem.SubItems.Add(filePath);
end;


procedure TMainFrm.addServer(ServerName, Ip: string; Port: Integer;
  Encrypt: Boolean);
var
  serverAddr: PServerAddrInfo;
begin
  serverAddr:=New(PServerAddrInfo);
  serverAddr.ServerName:=ServerName;
  serverAddr.Ip:=Ip;
  serverAddr.Port:=Port;
  serverAddr.Encrypt:=Encrypt;
  ServerList.Add(serverAddr);
end;

function TMainFrm.AnalysisServer(context: string): Boolean;
var
  JsonObject: TJSONObject;
  SubJsonObj: TJSONArray;
  ServerJsonObj: TJSONObject;
  ServerName, Ip: string;
  Port: Integer;
  Encrypt: Boolean;
  I: Integer;
begin
  ServerList.Clear;
  JsonObject := TJSONObject.ParseJSONValue(TEncoding.UTF8.GetBytes(trim(context)), 0) as TJSONObject;
  SubJsonObj := JsonObject.getValue('Addrs') as TJSONArray;
  for I := 0 to SubJsonObj.size - 1 do
  begin
    ServerJsonObj := SubJsonObj.Get(I) as TJSONObject;

    ServerName:=ServerJsonObj.Get('Name').JsonString.ToString;
    Ip:=ServerJsonObj.Get('Ip').JsonString.ToString;
//    Port:=ServerJsonObj.Get('Ip').JsonValue.ToString;
//    Encrypt:=ServerJsonObj.Get('Encrypt').JsonValue.TryGetValue()
    addServer(ServerName, Ip, Port, Encrypt);
  end;
end;

procedure TMainFrm.BitBtn1Click(Sender: TObject);
begin
  AddFile('E:\tools\windows_10_professional_x64_v2020.iso');
end;

procedure TMainFrm.BitBtn2Click(Sender: TObject);
begin
  GetServer();
end;

procedure TMainFrm.BitBtn3Click(Sender: TObject);
begin
  openClient('0.0.0.0', '115.29.176.57', 41002, 'log', 'D:/projects/Chainware/最终产品/windows/lib/harq/harq_32.dll', '123456');
end;

procedure TMainFrm.clearServer;
begin

end;

procedure TMainFrm.DirButClick(Sender: TObject);
var
  Dir: string;
begin
  SelectDirectory('请您选择目录', '', Dir);
  LocalDirEdt.Text:=Dir;

  //绘制信息;
  drawLocalDir(Dir);
end;

procedure TMainFrm.drawLocalDir(localDir: string);
var
  sPath, sFile:string;
  pSearchRec: TSearchRec;
  NewItem: TListItem;
begin
  //遍历目录;
  if Copy(localDir, Length(localDir), 1) <> '\' then
    sPath := localDir + '\'
  else
    sPath := localDir;

  //清空原有;
  LocalListView.Clear;

  if FindFirst(sPath + '*.*', faAnyFile, pSearchRec) = 0 then
  begin

    repeat
      sFile := Trim(pSearchRec.Name);

      // 排除自身文件夹，与父文件夹
      if sFile = '.' then Continue;
      if sFile = '..' then Continue;
      sFile := sPath + pSearchRec.Name;

      if(pSearchRec.Attr and faDirectory) <> 0 then
      begin

        //这是目录;
        NewItem := LocalListView.Items.Add;
        NewItem.Caption:='目录';
        NewItem.SubItems.Add(sFile);
      end
      else if(pSearchRec.Attr and faAnyFile) = pSearchRec.Attr then
      begin

        //这是文件;
        NewItem := LocalListView.Items.Add;
        NewItem.Caption:='文件';
        NewItem.SubItems.Add(sFile);
      end;
    until FindNext(pSearchRec) <> 0;
    FindClose(pSearchRec);
  end;
end;

procedure TMainFrm.drawServer;
var
  I: Integer;
  serverAddr: PServerAddrInfo;
begin
  for I := 0 to ServerList.Count - 1 do
  begin
    serverAddr:=ServerList.Items[I];
    if (serverAddr <> nil) then
    begin
      ServerComboBox.AddItem(serverAddr.ServerName + '(' + serverAddr.Ip + ':'+ IntToStr(serverAddr.Port) + ')', TObject(serverAddr));
    end;
  end;

  if ServerComboBox.Items.Count > 0 then
  begin
    ServerComboBox.Text := ServerComboBox.Items[0];
  end;
end;

procedure TMainFrm.DrawSubItem(ALV: TListView; AItem: TListItem;
  ASubItem: Integer; APosition: Single; AMax, AStyle: Integer;
  AIsShowProgress: Boolean; ADrawColor, AFrameColor: TColor);
var
  PaintRect, r: TRect;
  i, iWidth, x, y: integer;
  S: string;
  function GetItemRect(LV_Handle, iItem, iSubItem: Integer): TRect;
  var
    Rect: TRect;
  begin
    ListView_GetSubItemRect(LV_Handle, iItem, iSubItem, LVIR_LABEL, @Rect);
    Result := Rect;
  end;
 begin
   with ALV do
   begin
     PaintRect := GetItemRect(ALV.Handle, AItem.Index, ASubItem);
     r := PaintRect;
     //这一段是算出百分比
     if APosition >= AMax then
       APosition := 100
     else
       if APosition <= 0 then
       begin
         APosition := 0
       end
       else
       begin
         //APosition := Round((APosition / AMax) * 100);
         APosition := (APosition / AMax) * 100;
       end;

     if (APosition = 0) and (not AIsShowProgress) then
     begin
       //如果是百分比是0，就直接显示空白
       Canvas.FillRect(r);
     end
     else
     begin
       //先直充背景色
       Canvas.FillRect(r);
       Canvas.Brush.Color := Color;
       //画一个外框
       InflateRect(r, -2, -2);
       Canvas.Brush.Color := AFrameColor; //$00002F00;
       Canvas.FrameRect(R);
       Canvas.Brush.Color := Color;
       InflateRect(r, -1, -1);
       InflateRect(r, -1, -1);
       //根据百分比算出要画的进度条内容宽度
       iWidth := R.Right - Round((R.Right - r.Left) * ((100 - APosition) / 100));
       case AStyle of
         0: //进度条类型，实心填充
           begin
             Canvas.Brush.Color := ADrawColor;
             r.Right := iWidth;
             Canvas.FillRect(r);
           end;
         1: //进度条类型，竖线填充
           begin
             i := r.Left;
             while i < iWidth do
             begin
               Canvas.Pen.Color := Color;
               Canvas.MoveTo(i, r.Top);
               Canvas.Pen.Color := ADrawColor;
               canvas.LineTo(i, r.Bottom);
               Inc(i, 3);
             end;
           end;
       end;

       //画好了进度条后，现在要做的就是显示进度数字了
       Canvas.Brush.Style := bsClear;
       S := Format('%2f%%', [APosition]);
       with PaintRect do
       begin
         x := Left + (Right - Left + 1 - Canvas.TextWidth(S)) div 2;
         y := Top + (Bottom - Top + 1 - Canvas.TextHeight(S)) div 2;
       end;
       SetBkMode(Canvas.handle, TRANSPARENT);
       Canvas.TextRect(PaintRect, x, y, S);
     end; // end of if (Prosition = 0) and (not IsShowProgress) then
     Canvas.Brush.Color := Color;
   end; // end of with LV do
end;

function TMainFrm.findServer(ServerName: string): PServerAddrInfo;
begin

end;

procedure TMainFrm.FormActivate(Sender: TObject);
begin
  if not TaskTimer.Enabled then
  begin
    TaskTimer.Enabled:=true;
  end;
  ServerList.Clear;
  testServer();
  drawServer();
end;

procedure TMainFrm.FormCreate(Sender: TObject);
begin
  ServerList:=TDoubleList.Create(nil);
end;

function TMainFrm.GetFileName(filePath: string): String;
begin
  Result:=ExtractFileName(filePath);
end;

function TMainFrm.GetFileSizeByName(filePath: string): Int64;
var
  handle: THandle;
  dwHigh,dwLow:DWORD;
begin
  dwHigh:=0;
  if FileExists(filePath) then
  begin
    handle:= FileOpen(filePath, fmOpenRead or fmShareDenyNone);
    dwLow:=GetFileSize(handle, @dwHigh);
    if (dwLow = $FFFFFFFF) and (GetLastError() <> NO_ERROR) then
      Result:=0
    else
      Result:= (dwHigh shl 32) + dwLow;
    FileClose(handle);
  end
  else
    Result := 0;
end;

function UrlDecode(const AStr: AnsiString): AnsiString;
var
  Sp, Rp, Cp: PAnsiChar;
  s: AnsiString;
begin
  SetLength(Result, Length(AStr));
  Sp := PAnsiChar(AStr);
  Rp := PAnsiChar(Result);
  Cp := Sp;
  while Sp^ <> #0 do
  begin
    case Sp^ of
    '+':
      Rp^ := ' ';
    '%':
    begin
      Inc(Sp);
      if Sp^ = '%' then
        Rp^ := '%'
      else
      begin
        Cp := Sp;
        Inc(Sp);
        if (Cp^ <> #0) and (Sp^ <> #0) then
        begin
          s := AnsiChar('$') + Cp^ + Sp^;
          Rp^ := AnsiChar(StrToInt(string(s)));
        end;
      end;
      Cp := Cp;
    end;
    else
      Rp^ := Sp^;
    end;
    Inc(Rp);
    Inc(Sp);
  end;
  SetLength(Result, Rp - PAnsiChar(Result));
end;

procedure TMainFrm.GetHttpClientRequestCompleted(const Sender: TObject;
  const AResponse: IHTTPResponse);
begin
  AnalysisServer(TNetEncoding.URL.UrlDecode(AResponse.ContentAsString(TEncoding.GetEncoding(65001))));
end;

function TMainFrm.GetServer: boolean;
var
  vHttp: TNetHTTPClient;
  vUTF8, vGBK: TStringStream;
begin
  vHttp := TNetHTTPClient.Create(nil);
  vUTF8 := TStringStream.Create('', TEncoding.GetEncoding(65001));
  vGBK := TStringStream.Create('', TEncoding.GetEncoding(936));

  //设置属性
  GetHttpClient.Asynchronous := true;
  GetHttpClient.ConnectionTimeout := 10000; // 10秒
  GetHttpClient.ResponseTimeout := 10000;   // 10秒
  GetHttpClient.AcceptCharSet := 'utf-8';
  GetHttpClient.AcceptEncoding := '65001';
  GetHttpClient.AcceptLanguage := 'zh-CN';
  GetHttpClient.ContentType := 'text/html';
  GetHttpClient.UserAgent := 'Embarcadero URI Client/1.0';

  //发出Get请求;
  GetHttpClient.Get('http://10.10.50.211:8086/stream_service/new_stream?url=http://cctvcnch5c.v.wscdns.com/live/cctv8_2/index.m3u8');
end;

procedure TMainFrm.openClient(bindIp, remoteIp: string; remotePort: Integer;
  logPath, harqPath, session: string);
begin
    expressHandle:=open_client( PAnsichar(AnsiString(bindIp)),
                                PAnsichar(AnsiString(remoteIp)),
                                remotePort,
                                PAnsichar(AnsiString(logPath)),
                                PAnsichar(AnsiString(harqPath)),
                                PAnsichar(AnsiString(session)),
                                true,
                                @OnExpressLogin,
                                @OnExpressProgress,
                                @OnExpressFinish,
                                @OnExpressDisconnect,
                                @OnExpressError);
end;


procedure TMainFrm.TaskTimerTimer(Sender: TObject);
begin
  //任务列表刷新

end;

procedure TMainFrm.TaskViewCustomDrawItem(Sender: TCustomListView;
  Item: TListItem; State: TCustomDrawState; var DefaultDraw: Boolean);
 var
   BoundRect, Rect: TRect;
   i: integer;
   TextFormat: Word;
   LV: TListView;
begin
   LV := TListView(Sender);
   BoundRect := Item.DisplayRect(drBounds);
   InflateRect(BoundRect, -1, 0);

   //这个地方你可以根据自己的要求设置成想要的颜色，实现突出显示
   LV.Canvas.Font.Color := clBtnText;

   //查看是否被选中
   if Item.Selected then
   begin
     if cdsFocused in State then
     begin
       LV.Canvas.Brush.Color := $00ECCCB9; // //clHighlight;
     end
     else
     begin
       LV.Canvas.Brush.Color := $00F8ECE5; //clSilver;
     end;
   end
   else
   begin
     if (Item.Index mod 2) = 0 then
       LV.Canvas.Brush.Color := clWhite
     else
       LV.Canvas.Brush.Color := $00F2F2F2;
   end;


   LV.Canvas.FillRect(BoundRect); // 初始化背景
   for i := 0 to LV.Columns.Count - 1 do
   begin

    //获取SubItem的Rect
     ListView_GetSubItemRect(LV.Handle, Item.Index, i, LVIR_LABEL, @Rect);
     case LV.Columns[i].Alignment of
       taLeftJustify:
         TextFormat := DT_LEFT;
       taRightJustify:
         TextFormat := DT_RIGHT;
       taCenter:
         TextFormat := DT_CENTER;
     else
       TextFormat := DT_CENTER;
     end;

     case i of
       0: //画Caption,0表示Caption，不是Subitem
       begin
         InflateRect(Rect, -(5 + 0), 0); //向后移3个像素,避免被后面画线框时覆盖
         DrawText(LV.Canvas.Handle, PCHAR(Item.Caption), Length(Item.Caption), Rect, DT_VCENTER or DT_SINGLELINE or DT_END_ELLIPSIS or TextFormat);
       end;
       1..MaxInt: //画SubItem[i]
         begin
           if (i - 1) = 2 then //显示状态条，本示例是第三栏显示，
           begin
             DrawSubItem(LV, Item, i, StrToFloatDef(Item.SubItems[i - 1], 0), 100, 0, True, clMedGray);
           end
           else
           begin
             //画SubItem的文字
             InflateRect(Rect, -2, -2);
             if i - 1 <= Item.SubItems.Count - 1 then
               DrawText(LV.Canvas.Handle, PCHAR(Item.SubItems[i - 1]), Length(Item.SubItems[i - 1]), Rect, DT_VCENTER or DT_SINGLELINE or DT_END_ELLIPSIS or TextFormat);
           end;
         end;
     end; //end case
   end; //end for
   LV.Canvas.Brush.Color := clWhite;
   if Item.Selected then //画选中条外框
   begin
     if cdsFocused in State then//控件是否处于激活状态
       LV.Canvas.Brush.Color := $00DAA07A // $00E2B598; //clHighlight;
     else
       LV.Canvas.Brush.Color := $00E2B598; //$00DAA07A // clHighlight;

     LV.Canvas.FrameRect(BoundRect); //
   end;

   DefaultDraw := False; //不让系统画了
   with Sender.Canvas do
     if Assigned(Font.OnChange) then
       Font.OnChange(Font);

end;

procedure TMainFrm.testServer;
begin
  addServer('张三', '127.0.0.1', 41002, false);
  addServer('李四', '127.0.0.1', 41002, false);
  addServer('王五', '127.0.0.1', 41002, false);
end;

function TMainFrm.ReDrawItem(AHwndLV: HWND; AItemIndex: integer): boolean;
begin
  Result := ListView_RedrawItems(AHwndLV, AItemIndex, AItemIndex);
end;

procedure TMainFrm.SetFileProgress(fileName: string; progress: Integer);
var
  I: Integer;
begin
  if TaskView.Items.Count > 0 then
  begin
    for I := 0 to TaskView.Items.Count - 1 do
    begin
      if TaskView.Items.Item[I].SubItems.Strings[0] = fileName then
      begin
        TaskView.Items.Item[I].SubItems.Strings[2] := intToStr(progress);
      end;
    end;
  end;
end;

end.
