program file_express;

uses
  Vcl.Forms,
  Unit1 in 'Unit1.pas' {MainFrm},
  Vcl.Themes,
  Vcl.Styles;

{$R *.res}

begin
  Application.Initialize;
  Application.MainFormOnTaskbar := True;
  Application.CreateForm(TMainFrm, MainFrm);
  Application.Run;
end.