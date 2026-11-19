// Test-only screenshot evidence for left/right preview rotation.
// This does not read application memory or modify the source files.
using System;
using System.Drawing;
using System.Collections.Generic;
public static class DcmRotationProbe {
    public sealed class ImageResult {
        public int Orientation; public int Matches; public int DrawX,DrawY,DrawWidth,DrawHeight;
    }
    public sealed class PanelResult {
        public string Color; public int Samples; public int Left,Top,Right,Bottom;
        public double Ratio;
    }
    static readonly Color[] Palette={Color.FromArgb(20,160,190),Color.FromArgb(230,140,30),Color.FromArgb(140,65,195),Color.FromArgb(50,170,70)};
    static bool Near(Color a,Color b,int tolerance) {
        return Math.Abs(a.R-b.R)<=tolerance && Math.Abs(a.G-b.G)<=tolerance && Math.Abs(a.B-b.B)<=tolerance;
    }
    public static ImageResult CheckImage(string path,int left,int top,DcmImageProbe.View v,int w,int h,int turns) {
        if(turns<0 || turns>3)throw new Exception("Invalid orientation");
        int displayW=(turns%2==0)?w:h,displayH=(turns%2==0)?h:w;
        double scale=Math.Min((double)(v.Width-8)/displayW,(double)(v.Height-8)/displayH);
        int dw=(int)(displayW*scale),dh=(int)(displayH*scale);
        int dx=(v.Width-dw)/2,dy=(v.Height-dh)/2;
        int[][] quadrants={new[]{0,1,2,3},new[]{2,0,3,1},new[]{3,2,1,0},new[]{1,3,0,2}};
        var result=new ImageResult{Orientation=turns,DrawX=dx,DrawY=dy,DrawWidth=dw,DrawHeight=dh};
        using(var bitmap=new Bitmap(path)) {
            for(int q=0;q<4;q++) {
                for(int n=0;n<3;n++) {
                    double fx=(q%2==0?.23:.77)+((n-1)*.04);
                    double fy=(q<2?.22:.78);
                    int lx=dx+(int)(fx*dw),ly=dy+(int)(fy*dh);
                    int x=v.Left-left+(int)((double)lx*v.PhysicalWidth/v.Width);
                    int y=v.Top-top+(int)((double)ly*v.PhysicalHeight/v.Height);
                    if(x<0 || y<0 || x>=bitmap.Width || y>=bitmap.Height)throw new Exception("Rotated sample escaped capture.");
                    Color actual=bitmap.GetPixel(x,y);
                    Color expect=Palette[quadrants[turns][q]];
                    if(!Near(actual,expect,42))throw new Exception("ROTATION_PIXEL_MISMATCH turns="+turns+" quadrant="+q+" actual="+actual+" expected="+expect+" xy="+x+","+y);
                    result.Matches++;
                }
            }
        }
        if(result.Matches!=12)throw new Exception("Incomplete rotated image pixel evidence");
        return result;
    }
    public static PanelResult PdfPanel(string path,int left,int top,DcmImageProbe.View v,string color) {
        var panel=new PanelResult{Color=color,Left=int.MaxValue,Top=int.MaxValue,Right=-1,Bottom=-1};
        using(var image=new Bitmap(path)) {
            for(int ly=40;ly<v.Height-5;ly+=3)for(int lx=5;lx<v.Width-5;lx+=3){
                int x=v.Left-left+(int)((double)lx*v.PhysicalWidth/v.Width);
                int y=v.Top-top+(int)((double)ly*v.PhysicalHeight/v.Height);
                if(x<0||y<0||x>=image.Width||y>=image.Height)continue;
                Color p=image.GetPixel(x,y);
                bool match=color=="teal"?(p.G>p.R+40 && p.B>p.R+50 && p.G>80):
                             (p.R>p.G+70 && p.G>p.B+30 && p.R>150);
                if(!match)continue;
                panel.Samples++;
                panel.Left=Math.Min(panel.Left,lx);panel.Top=Math.Min(panel.Top,ly);
                panel.Right=Math.Max(panel.Right,lx);panel.Bottom=Math.Max(panel.Bottom,ly);
            }
        }
        if(panel.Samples<100 || panel.Right<=panel.Left || panel.Bottom<=panel.Top)throw new Exception("PDF color panel not rendered for "+color);
        panel.Ratio=(double)(panel.Right-panel.Left)/(panel.Bottom-panel.Top);
        return panel;
    }
}
