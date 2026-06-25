%% Copyright (C) 2010-2025 by Martin Scharrer <martin.scharrer@web.de>
%% ---------------------------------------------------------------------------
%% This work may be distributed and/or modified under the
%% conditions of the LaTeX Project Public License, either version 1.3
%% of this license or (at your option) any later version.
%% The latest version of this license is in
%%   http://www.latex-project.org/lppl.txt
%% and version 1.3 or later is part of all distributions of LaTeX
%% version 2005/12/01 or later.
%%
%% This work has the LPPL maintenance status `maintained'.
%%
%% The Current Maintainer of this work is Martin Scharrer.
%%
%% This work consists of the files standalone.dtx and standalone.ins
%% and the derived filebase standalone.sty, standalone.cls and
%% standalone.cfg.
%%
\NeedsTeXFormat{LaTeX2e}
\ProvidesClass{standalone}[%
    2025/02/22
    v1.5a
    Class to compile TeX sub-files standalone]
\newif\ifstandalone
\standalonetrue
\newif\ifstandalonebeamer
\standalonebeamerfalse
\let\onlyifstandalone\@firstofone
\let\IfStandalone\@firstoftwo
\def\sa@border@left{0.50001bp}
\let\sa@border@right\sa@border@left
\let\sa@border@top\sa@border@left
\let\sa@border@bottom\sa@border@left
\def\rem@bp#1bp\relax#2\@nnil{#1}%
\def\default@bp#1#2{%
    \begingroup
    \afterassignment\remove@to@nnil
    \dimen@ #2bp\relax\@nnil
    \expandafter
    \endgroup
    \expandafter
    \def\expandafter#1\expandafter{\the\dimen@}%
}
\def\sa@readborder#1 #2 #3 #4 #5\@nnil{%
    \ifx\\#2#3#4\\%
        \default@bp\sa@border@left{#1}%
        \let\sa@border@right\sa@border@left
        \let\sa@border@top\sa@border@left
        \let\sa@border@bottom\sa@border@left
    \else
    \ifx\\#4\\%
        \default@bp\sa@border@left{#1}%
        \let\sa@border@right\sa@border@left
        \default@bp\sa@border@top{#2}%
        \let\sa@border@bottom\sa@border@top
    \else
        \default@bp\sa@border@left{#1}%
        \default@bp\sa@border@bottom{#2}%
        \default@bp\sa@border@right{#3}%
        \default@bp\sa@border@top{#4}%
    \fi\fi
}%
\expandafter\ifx\csname ShellEscape\endcsname\relax
    \IfFileExists{shellesc.sty}{
        \RequirePackage{shellesc}
        \@ifpackagelater{shellesc}{2016/04/29}{
        }{
            \protected\def\ShellEscape{\immediate\write18 }
        }
    }{
        \protected\def\ShellEscape{\immediate\write18 }
    }
\fi
\expandafter\ifx\csname ifluatex\endcsname\relax
    \IfFileExists{ifluatex.sty}{\@firstoftwo}{\@secondoftwo}{%
        \RequirePackage{ifluatex}
    }{
        \begingroup
        \expandafter\ifx\csname directlua\endcsname\relax
            \endgroup
            \expandafter\let\csname ifluatex\expandafter\endcsname\csname iffalse\endcsname
        \else
            \endgroup
            \expandafter\let\csname ifluatex\expandafter\endcsname\csname iftrue\endcsname
        \fi
    }
\fi
\expandafter\ifx\csname ifpdf\endcsname\relax
    \IfFileExists{ifpdf.sty}{\@firstoftwo}{\@secondoftwo}{%
        \RequirePackage{ifpdf}
    }{
        \begingroup
        \expandafter\ifx\csname pdfoutput\endcsname\relax
            \endgroup
            \expandafter\let\csname ifpdf\expandafter\endcsname\csname iffalse\endcsname
        \else
            \endgroup
            \ifnum\pdfoutput<1
                \expandafter\let\csname ifpdf\expandafter\endcsname\csname iffalse\endcsname
            \else
                \expandafter\let\csname ifpdf\expandafter\endcsname\csname iftrue\endcsname
            \fi
        \fi
    }
\fi
\expandafter\ifx\csname ifxetex\endcsname\relax
    \IfFileExists{ifxetex.sty}{\@firstoftwo}{\@secondoftwo}{%
        \RequirePackage{ifxetex}
    }{
        \begingroup
        \expandafter\ifx\csname XeTeXrevision\endcsname\relax
            \endgroup
            \expandafter\let\csname ifxetex\expandafter\endcsname\csname iffalse\endcsname
        \else
            \endgroup
            \expandafter\let\csname ifxetex\expandafter\endcsname\csname iftrue\endcsname
        \fi
    }
\fi
\let\sa@classoptionslist\@classoptionslist
\RequirePackage{xkeyval}
\newif\ifsa@preview
\newif\ifsa@crop
\newif\ifsa@multi
\newif\ifsa@multido
\newif\ifsa@varwidth
\newif\ifsa@ignorerest
\newif\ifsa@ignoreempty
\newif\ifsa@tikz
\newif\ifsa@pstricks
\newif\ifsa@convert
\newif\ifsa@float
\newif\ifsa@math
\newif\ifsa@convert@perpage
\let\sa@beamertrue\standalonebeamertrue
\let\sa@beamerfalse\standalonebeamerfalse
\def\sa@clsoption{%
    \define@key{standalone.cls}%
}
\sa@clsoption{border}{%
    \sa@readborder#1 {} {} {} {} \@nnil
}
\sa@clsoption{margin}{%
    \sa@readborder#1 {} {} {} {} \@nnil
}
\def\sa@boolean#1#2{%
    \sa@boolorvalue{#1}{#2}%
        {\ClassError{standalone}{Invalid value '#2' for boolean key '#1'}{}}%
}
\def\sa@boolorvalue#1#2{%
    \begingroup
    \edef\@tempa{#2}%
    \def\@tempb{true}%
    \ifx\@tempa\@tempb
        \endgroup
        \csname sa@#1true\endcsname
        \expandafter\@gobble
    \else
    \def\@tempb{false}%
    \ifx\@tempa\@tempb
        \endgroup
        \csname sa@#1false\endcsname
        \expandafter\expandafter
        \expandafter\@gobble
    \else
        \endgroup
        \expandafter\expandafter
        \expandafter\@firstofone
    \fi\fi
}
\sa@clsoption{preview}[true]{%
    \sa@boolean{preview}{#1}%
    \ifsa@preview
        \setkeys{standalone.cls}{crop=false,float=false}%
    \fi
}
\sa@previewtrue
\sa@clsoption{crop}[true]{%
    \sa@boolean{crop}{#1}%
    \ifsa@crop
        \setkeys{standalone.cls}{preview=false,float=false}%
    \fi
}
\sa@clsoption{ignorerest}[true]{%
    \sa@boolean{ignorerest}{#1}%
}
\sa@clsoption{ignoreempty}[true]{%
    \sa@boolean{ignoreempty}{#1}%
}
\sa@clsoption{multi}[true]{%
    \sa@boolorvalue{multi}{#1}{\sa@multitrue\AtBeginDocument{\standaloneenv{#1}}}%
}
\sa@clsoption{multido}[true]{%
    \sa@boolean{multido}{#1}%
    \ifsa@multido
        \setkeys{standalone.cls}{multi=samultido}%
    \fi
}
\sa@clsoption{math}[true]{%
    \sa@boolean{math}{#1}%
    \ifsa@math
        \setkeys{standalone.cls}{multi=true,ignoreempty=true,border=0.50001bp}%
    \fi
}
\AtBeginDocument{\ifsa@math\sa@math\fi}
\def\sa@math{%
    \standaloneenv{math}%
    \def\({\begingroup\math}%
    \def\){\endmath\endgroup}%
    \def\[{\(\displaystyle}%
    \def\]{\)}%
    \def\displaymath{\math\displaystyle}%
    \def\enddisplaymath{\endmath}%
    \newcommand*\multimathsep{%
        \endmath
        \math
        \let\\\multimathsep
    }%
    \newenvironment{multimath}{%
        \math
        \let\\\multimathsep
    }{%
        \endmath
    }%
    \newcommand*\multidisplaymathsep{%
        \endmath
        \math\displaystyle
        \let\\\multidisplaymathsep
    }%
    \newenvironment{multidisplaymath}{%
        \math\displaystyle
        \let\\\multidisplaymathsep
    }{%
        \endmath
    }%
}
\sa@clsoption{varwidth}[true]{%
    \sa@boolorvalue{varwidth}{#1}{\sa@varwidthtrue\def\sa@width{#1}}%
    \ifsa@varwidth
        \def\sa@varwidth{\varwidth{\sa@width}}%
        \def\sa@endvarwidth{\endvarwidth}%
    \else
        \let\sa@varwidth\@empty
        \let\sa@endvarwidth\@empty
    \fi
}
\let\sa@varwidth\@empty
\let\sa@endvarwidth\@empty
\sa@clsoption{tikz}[true]{%
    \sa@boolean{tikz}{#1}%
    \ifsa@tikz
        \setkeys{standalone.cls}{multi=tikzpicture,varwidth=false}%
    \fi
}
\sa@clsoption{pstricks}[true]{%
    \sa@boolean{pstricks}{#1}%
    \ifsa@pstricks
        \setkeys{standalone.cls}{multi=pspicture,varwidth=false}%
    \fi
}
\sa@clsoption{beamer}[true]{%
    \sa@boolean{beamer}{#1}%
    \ifstandalonebeamer
        \def\sa@class{beamer}%
        \setkeys{standalone.cls}{preview=false,crop=false,varwidth=false}%
    \else
        \begingroup
        \def\@tempa{beamer}%
        \ifx\@tempa\sa@class
            \endgroup
            \def\sa@class{article}%
        \else
            \endgroup
        \fi
    \fi
}
\sa@clsoption{class}{%
    \def\sa@class{#1}%
}
\def\sa@class{article}
\sa@clsoption{float}[true]{%
    \sa@boolean{float}{#1}%
    \ifsa@float
        \let\@float\sa@origfloat
        \let\end@float\sa@origendfloat
    \else
        \ifx\@float\sa@nofloat\else
            \let\sa@origfloat\@float
        \fi
        \ifx\end@float\sa@endnofloat\else
            \let\sa@origendfloat\end@float
        \fi
        \let\@float\sa@nofloat
        \let\end@float\sa@endnofloat
    \fi
}
\def\sa@nofloat#1{%
    \def\@captype{#1}%
    \trivlist\item[]%
    \@ifnextchar[{%
        \begingroup
        \def\@tempa[####1]{%
            \endgroup
        }\@tempa
    }{}%
}
\def\sa@endnofloat{%
    \endtrivlist
}
\sa@clsoption{convert}[]{%
    \setkeys{standalone.cls/convert}{true,#1}%
}
\sa@clsoption{disable@convert}[]{%
    \typeout{Disable conversion}
    \sa@convertfalse
    \let\sa@converttrue\relax
}
\def\sa@convertoption{%
    \define@key{standalone.cls/convert}%
}
\def\sa@convertvar#1#2{%
    \define@key{standalone.cls/convert}{#1}{%
        \@namedef{sa@convert@#1}{##1}%
    }%
    \@namedef{sa@convert@#1}{#2}%
}
\sa@convertoption{true}[]{%
    \sa@converttrue
}
\sa@convertoption{false}[]{%
    \sa@convertfalse
}
\sa@convertoption{realmainfile}[]{%
    \RequirePackage{currfile-abspath}%
    \getmainfile
    \let\sa@convert@mainfile\themainfile
}
\sa@convertoption{png}[]{%
    \setkeys{standalone.cls/convert}{true,outext={.png}}%
}
\sa@clsoption{png}[]{%
    \setkeys{standalone.cls/convert}{png,#1}%
}
\sa@convertoption{jpg}[]{%
    \setkeys{standalone.cls/convert}{true,outext={.jpg}}%
}
\sa@clsoption{jpg}[]{%
    \setkeys{standalone.cls/convert}{jpg,#1}%
}
\sa@convertoption{gif}[]{%
    \setkeys{standalone.cls/convert}{true,imagemagick,outext={.gif}}%
}
\sa@clsoption{gif}[]{%
    \setkeys{standalone.cls/convert}{gif,#1}%
}
\sa@convertoption{svg}[]{%
    \setkeys{standalone.cls/convert}{true,inkscape,outext={.svg}}%
}
\sa@clsoption{svg}[]{%
    \setkeys{standalone.cls/convert}{svg,#1}%
}
\sa@convertoption{emf}[]{%
    \setkeys{standalone.cls/convert}{true,inkscape,outext={.emf}}%
}
\sa@clsoption{emf}[]{%
    \setkeys{standalone.cls/convert}{emf,#1}%
}
\sa@convertoption{onfailure}{%
    \begingroup
    \edef\@tempa{#1}%
    \def\@tempb{error}%
    \ifx\@tempa\@tempb
        \endgroup
        \let\sa@convert@failuremsg\ClassError
    \else
    \def\@tempb{warning}%
    \ifx\@tempa\@tempb
        \endgroup
        \let\sa@convert@failuremsg\ClassWarning
    \else
    \def\@tempb{info}%
    \ifx\@tempa\@tempb
        \endgroup
        \let\sa@convert@failuremsg\ClassInfo
    \else
    \def\@tempb{ignore}%
    \ifx\@tempa\@tempb
        \endgroup
        \def\sa@convert@failuremsg##1##2##3{}%
        \let\sa@convert@notfoundmsg\@gobbletwo
    \else
        \let\on@line\@empty
        \ClassError{standalone}{Invalid value '\@tempa' for the 'onfailure' option.\MessageBreak
                                Valid values: 'error', 'warning', 'info', 'ignore'}{}%
        \endgroup
    \fi\fi\fi\fi
}
\let\sa@convert@failuremsg\ClassWarning
\let\sa@convert@notfoundmsg\ClassWarning
\sa@convertoption{perpage}[true]{%
    \sa@boolean{convert@perpage}{#1}%
}
\sa@convertoption{defgsdevice}{%
    \sa@defgsdevice#1\relax\relax
}
\def\sa@defgsdevice#1#2{%
    \@namedef{sa@gsdevice@#1}{#2}%
}
\@namedef{sa@gsdevice@.jpg}{jpeg}%
\@namedef{sa@gsdevice@.png}{png16m}%
\sa@convertoption{command}{%
    \def\sa@convert@command{#1}%
}
\sa@convertoption{pdf2svg}[]{%
    \def\sa@convert@command{pdf2svg \infile\space\outfile}%
    \sa@convertvar{outext}{.svg}
}
\sa@convertoption{imagemagick}[]{%
    \def\sa@convert@command{\convertexe\space
        -density \density\space
        -units PixelsPerInch
        \infile\ifsa@convert@perpage[\prevpagenum]\fi\space
        -scene 1
        \ifx\size\empty\else -resize \size\fi\space
        -quality 90 \outfile}%
}
\sa@convertoption{ghostscript}[]{%
    \def\sa@convert@command{\gsexe\space -dSAFER -dBATCH -dNOPAUSE
        \ifsa@convert@perpage-sPageList=\pagenum\fi\space
        -sDEVICE=\gsdevice\space
        -r\density\space
        -sOutputFile=\outfile\space \infile
    }%
}
\sa@convertoption{ghostscript32}[]{%
    \setkeys{standalone.cls/convert}{ghostscript,gsexe=\@nameuse{sa@convert@gsexe32}}%
}
\sa@convertoption{ghostscript64}[]{%
    \setkeys{standalone.cls/convert}{ghostscript,gsexe=\@nameuse{sa@convert@gsexe64}}%
}
\sa@convertoption{inkscape}[]{%
    \def\sa@convert@command{\inkscapeexe\space
        --pdf-poppler \sa@inkscape@bgcolor\space
        --export-type="\expandafter\@gobble\outext"
        --pdf-page=\pagenum\space
        -o \outfile\space \infile}
    \sa@convert@perpagetrue
    \def\sa@multi@pagemark{\pagenum}%
}
\def\sa@inkscape@bgcolor{\ifx\sa@convert@bgcolor\@empty\else--actions="export-background:\sa@convert@bgcolor"\fi}
\sa@convertvar{latexoptions}{ -shell-escape \ifx\outputdir\empty\else-output-directory \sa@convert@quote\outputdir\sa@convert@quote\fi\space}
\sa@convertvar{subjobname}{\jobname}
\sa@convertvar{mainfile}{\jobname}
\sa@convertvar{quote}{}
\let\sa@convert@quote\relax
\sa@convertvar{size}{}
\sa@convertvar{inname}{\subjobname}
\sa@convertvar{infile}{\indir\inname\inext}
\sa@convertvar{outputdir}{}
\sa@convertvar{indir}{\outputdir}
\sa@convertvar{outdir}{\outputdir}
\sa@convertvar{outext}{.png}
\sa@convertvar{outname}{\inname}
\sa@convertvar{outfile}{\outdir\outname\ifsa@multi\sa@multi@pagemark\fi\outext}
\def\sa@multi@pagemark{-\percent0d}
\def\sa@convert@pagenum{1}
\sa@convertvar{density}{300}
\sa@convertvar{bgcolor}{}
\sa@convertvar{gsdevice}{%
    \expandafter\ifx\csname sa@gsdevice@\outext\endcsname\relax
        \expandafter\@gobble\outext
    \else
        \csname sa@gsdevice@\outext\endcsname
    \fi
}
\ifluatex
    \sa@convertvar{latex}{lualatex}
    \sa@convertvar{inext}{.pdf}
    \sa@convertvar{precommand}{}
    \setkeys{standalone.cls/convert}{ghostscript}
\else
\ifpdf
    \sa@convertvar{latex}{pdflatex}
    \sa@convertvar{inext}{.pdf}
    \sa@convertvar{precommand}{}
    \setkeys{standalone.cls/convert}{ghostscript}
\else
\ifxetex
    \sa@convertvar{latex}{xelatex}
    \sa@convertvar{inext}{.pdf}
    \sa@convertvar{precommand}{}
    \setkeys{standalone.cls/convert}{ghostscript}
\else
    \sa@convertvar{latex}{latex}
    \sa@convertvar{inext}{.ps}
    \sa@convertvar{precommand}{dvips \subjobname.dvi}
    \setkeys{standalone.cls/convert}{ghostscript}
\fi\fi\fi
\begingroup
\ifluatex
  \csname @tempswa\directlua{
      if os.type == "windows" then
        tex.sprint("true")
      else
        tex.sprint("false")
      end
    }\endcsname
\else
    \IfFileExists{/dev/null}{\@tempswafalse}{\@tempswatrue}%
\fi
\if@tempswa
    \endgroup
    \@namedef{sa@convert@gsexe32}{gswin32c}
    \@namedef{sa@convert@gsexe64}{gswin64c}
\else
    \endgroup
    \@namedef{sa@convert@gsexe32}{gs}
    \@namedef{sa@convert@gsexe64}{gs}
\fi
\sa@convertvar{gsexe}{\@nameuse{sa@convert@gsexe64}}
\sa@convertvar{convertexe}{magick convert}
\sa@convertvar{inkscapeexe}{inkscape}
\newcommand*\standaloneenv[1]{%
    \begingroup
    \edef\@tempa{\endgroup\noexpand\@for\noexpand\@tempa:=\zap@space#1 \@empty}%
    \@tempa\do{\expandafter\@standaloneenv\expandafter{\@tempa}}%
    \setkeys{standalone.cls}{multi}%
}
\@onlypreamble\standaloneenv
\newcommand*{\standaloneconfig}{\setkeys{standalone.cls}}
\let\@standaloneenv\@gobble
\newcount\sa@internal
\newcounter{sapage}
\let\standalone\empty
\let\endstandalone\relax
\def\sa@width{\linewidth}
\InputIfFileExists{standalone.cfg}{}{}
\begingroup
\def\@tempa{\endgroup\setkeys*{standalone.cls}}
\expandafter\expandafter\expandafter\@tempa
\expandafter\expandafter\expandafter{\csname opt@standalone.cls\endcsname}
\let\@classoptionslist\XKV@rm
\disable@keys{standalone.cls}{crop,preview,class,beamer,ignorerest}
\AtBeginDocument{%
    \disable@keys{standalone.cls}{multi}%
}
\expandafter\expandafter\expandafter\LoadClass
\expandafter\expandafter\expandafter[%
\expandafter\@classoptionslist
\expandafter]\expandafter{\sa@class}
\ifsa@ignorerest
    \def\sa@startignore{\sa@boxit}
\else
    \let\sa@startignore\relax
\fi
\ifsa@ignorerest
    \def\sa@stopignore{\endsa@boxit}
\else
    \let\sa@stopignore\relax
\fi
\ifsa@multido
\RequirePackage{multido}
\let\sa@orig@multido@\multido@
\renewcommand{\multido@}[6]{%
  \sa@stopignore
  \sa@orig@multido@{#1}{#2}{#3}{#4}{#5}{%
    \sa@startignore
    \begin{samultido}%
      \let\multido@\sa@orig@multido@
      #6%
    \end{samultido}%
    \sa@stopignore
  }%
  \sa@startignore
}
\fi
\ifluatex
\RequirePackage{luatex85}
\RequirePackage{pdftexcmds}
\fi
\ifsa@convert
\ifx\sa@convert@quote\relax
\begingroup
\@tempswafalse
\expandafter\ifx\csname pdftexbanner\endcsname\relax
    \@tempswatrue
\else
\def\MiKTeX{MiKTeX}
\@onelevel@sanitize\MiKTeX
\expandafter\def\expandafter\testmiktex\expandafter#\expandafter1\MiKTeX#2\relax{%
        \ifx\empty#2\empty
             \@tempswafalse
        \else
             \@tempswatrue
        \fi
}
\expandafter\expandafter
\expandafter\testmiktex\expandafter\pdftexbanner\MiKTeX\relax\relax

\fi
\expandafter
\endgroup
\if@tempswa
\def\sa@convert@quote{"}
\else
\def\sa@convert@quote{'}
\fi
\fi
\fi
\ifsa@varwidth
    \RequirePackage{varwidth}
\fi
\ifsa@tikz
    \RequirePackage{tikz}
\fi
\ifsa@pstricks
    \RequirePackage{pstricks}
\fi
\ifsa@preview
\RequirePackage{preview}
\ifsa@multi\else
    \@ifundefined{endstandalone}{%
        \renewenvironment{standalone}
            {\preview\sa@varwidth}
            {\sa@endvarwidth\endpreview}
    }{}% TODO: Add info message?
\fi
\def\PreviewBbAdjust{-\sa@border@left\space -\sa@border@bottom\space \sa@border@right\space \sa@border@top}%
\def\@standaloneenv#1{%
    \expandafter\ifx\csname sa@orig@#1\endcsname\relax
        \expandafter\let\csname sa@orig@#1\expandafter\endcsname\csname #1\endcsname
        \expandafter\let\csname sa@orig@end#1\expandafter\endcsname\csname end#1\endcsname
    \fi
    \expandafter\def\csname #1\endcsname{%
        \ifnum\sa@internal=0
            \addtocounter{sapage}\@ne
            \preview
            \sa@varwidth
        \fi
        \advance\sa@internal\@ne
        \csname sa@orig@#1\endcsname
    }%
    \expandafter\def\csname end#1\endcsname{%
        \csname sa@orig@end#1\endcsname
        \advance\sa@internal\m@ne
        \ifnum\sa@internal=0
            \sa@endvarwidth
            \endpreview
        \fi
    }%
}%
\fi
\ifsa@crop
\newbox\sa@box
\pagestyle{empty}
\hoffset=-72.27pt
\voffset=-72.27pt
\topmargin=0pt
\headheight=0pt
\headsep=0pt
\marginparsep=0pt
\marginparwidth=0pt
\footskip=0pt
\marginparpush=0pt
\oddsidemargin=0pt
\evensidemargin=0pt
\topskip=0pt
\textheight=\maxdimen
\def\sa@boxit{%
    \setbox\sa@box\hbox\bgroup\color@setgroup\sa@varwidth
}%
\def\endsa@boxit{%
    \sa@endvarwidth\color@endgroup\egroup
}%
\renewenvironment{standalone}{%
    \ifsa@multi
        \sa@startignore
    \else
        \sa@boxit
    \fi
}{%
    \ifsa@multi
        \sa@stopignore
    \else
        \endsa@boxit
        \sa@handlebox
    \fi
}
\ifsa@multi\else
    \sa@ignorerestfalse
\fi
\ifsa@ignorerest
    \def\@standaloneenv#1{%
        \expandafter\ifx\csname sa@orig@#1\endcsname\relax
            \expandafter\let\csname sa@orig@#1\expandafter\endcsname\csname #1\endcsname
            \expandafter\let\csname sa@orig@end#1\expandafter\endcsname\csname end#1\endcsname
        \fi
        \expandafter\def\csname #1\endcsname{%
            \ifnum\sa@internal=0
                \addtocounter{sapage}\@ne
                \edef\@tempa{\endgroup
                    \noexpand\endsa@boxit
                    \begingroup
                    \def\noexpand\@currenvir{\@currenvir}%
                    \def\noexpand\@currenvline{\@currenvline}%
                }%
                \@tempa
                \sa@boxit
            \fi
            \advance\sa@internal\@ne
            \csname sa@orig@#1\endcsname
        }%
        \expandafter\def\csname end#1\endcsname{%
            \csname sa@orig@end#1\endcsname
            \advance\sa@internal\m@ne
            \ifnum\sa@internal=0
                \endsa@boxit
                \sa@handlebox
                \aftergroup\sa@boxit
            \fi
        }%
    }%
\else
    \def\@standaloneenv#1{%
        \expandafter\ifx\csname sa@orig@#1\endcsname\relax
            \expandafter\let\csname sa@orig@#1\expandafter\endcsname\csname #1\endcsname
            \expandafter\let\csname sa@orig@end#1\expandafter\endcsname\csname end#1\endcsname
        \fi
        \expandafter\def\csname #1\endcsname{%
            \ifnum\sa@internal=0
                \addtocounter{sapage}\@ne
                \sa@boxit
            \fi
            \advance\sa@internal\@ne
            \csname sa@orig@#1\endcsname
        }%
        \expandafter\def\csname end#1\endcsname{%
            \csname sa@orig@end#1\endcsname
            \advance\sa@internal\m@ne
            \ifnum\sa@internal=0
                \endsa@boxit
                \sa@handlebox
            \fi
        }%
    }%
\fi
\def\sa@handlebox{%
    \ifcase
        0%
        \ifsa@ignoreempty
            \ifdim\wd\sa@box=\z@
            \ifdim\ht\sa@box=\z@
            \ifdim\dp\sa@box=\z@
                1%
            \fi\fi\fi
        \fi
    \relax
    \sbox\sa@box{%
        \hskip\sa@border@left
        \@tempdima=\ht\sa@box
        \advance\@tempdima\sa@border@top\relax
        \ht\sa@box=\@tempdima
        \@tempdima=\dp\sa@box
        \advance\@tempdima\sa@border@bottom\relax
        \dp\sa@box=\@tempdima
        \raise\dp\sa@box
            \box\sa@box
        \hskip\sa@border@right
    }%
    \sa@placebox
    \fi
}
\ifcase0%
    \ifpdf\else\ifluatex\else\ifxetex\else 1\fi\fi\fi
  \relax
    \def\sa@placebox{%
        \newpage
        \ifluatex
          \ifnum\luatexversion<85
            % LuaLaTeX < 0.85 does define the PDF-specific globals.
            \global\pdfpagewidth=\wd\sa@box
            \global\pdfpageheight=\ht\sa@box
          \else
            % LuaLaTeX >= 0.85 doesn't define the PDF-specific globals.
            \global\pagewidth=\wd\sa@box
            \global\pageheight=\ht\sa@box
          \fi
        \else
          % Not LuaLaTeX at all, so still need this.
          \global\pdfpagewidth=\wd\sa@box
          \global\pdfpageheight=\ht\sa@box
        \fi
        \global\paperwidth=\wd\sa@box
        \global\paperheight=\ht\sa@box
        \special{papersize=\the\paperwidth,\the\paperheight}
        \global\hsize=\wd\sa@box
        \global\vsize=\ht\sa@box
        \global\@colht=\ht\sa@box
        \global\@colroom=\ht\sa@box
        \noindent\usebox\sa@box
        \newpage
    }
  \else
    \def\sa@placebox{%
        \global\paperwidth=\wd\sa@box
        \global\paperheight=\ht\sa@box
        \special{papersize=\the\paperwidth,\the\paperheight}
        \global\@colht=\maxdimen
        \global\@colroom=\maxdimen
        \global\hsize=\maxdimen
        \global\vsize=\maxdimen
        \sa@papersize
        \ifsa@multi
        \begingroup
        \@tempdima0.99626\paperwidth
        \@tempdimb0.99626\paperheight
        \edef\@tempc{\strip@pt\@tempdima}%
        \edef\@tempd{\strip@pt\@tempdimb}%
        \advance\@tempdima by .998pt
        \advance\@tempdimb by .998pt
        \def\strip@float##1.##2\relax{##1}%
        \edef\@tempa{\expandafter\strip@float\the\@tempdima\relax}%
        \edef\@tempb{\expandafter\strip@float\the\@tempdimb\relax}%
        \special{ps::%
            \@percentchar\@percentchar PageBoundingBox: 0 0 \@tempa\space\@tempb^^J%
            \@percentchar\@percentchar HiResPageBoundingBox: 0 0 \@tempc\space\@tempd^^J%
            \@percentchar\@percentchar BeginPageSetup^^J%
            << /PageSize [\@tempc\space\@tempd]
            >> setpagedevice^^J%<<
            0 0 bop^^J%
            \@percentchar\@percentchar EndPageSetup}%
        \endgroup
        \fi
        \topskip=0pt
        \noindent\sa@ps@content
        \newpage
    }
\def\sa@ps@content{%
    \noindent\usebox\sa@box
    \global\def\sa@ps@content{%
        \@tempdima\sa@yoffset
        \advance\@tempdima-\topskip
        \dp\sa@box\z@
        \ht\sa@box\z@
        \noindent\lower\@tempdima\copy\sa@box
    }%
}
\def\sa@papersize{%
    \global\let\sa@papersize\relax
    \global\sa@yoffset=\paperheight
    \global\setbox\@begindvibox\vbox{%
        \special{papersize=\the\paperwidth,\the\paperheight}%
        \special{ps::%
            \@percentchar\@percentchar HiResBoundingBox: 0 0 \the\paperwidth\space\the\paperheight^^J%
        }%
        \unvbox\@begindvibox
        \special{papersize=\the\paperwidth,\the\paperheight}%
    }%
}
\newlength\sa@yoffset
\fi
\fi
\ifstandalonebeamer
\newenvironment{standaloneframe}{%
  \@ifnextchar<%
    {\@standaloneframe}%
    {\@@standaloneframe{}}%
}{\end{frame}}%
\def\@standaloneframe<#1>{%
    \@@standaloneframe{<#1>}%
}
\def\@@standaloneframe#1{%
  \@ifnextchar[%]
    {\@@@standaloneframe{#1}}%
    {\@@@standaloneframe{#1}[]}%
}%
\def\@@@standaloneframe#1[{%
  \@ifnextchar<%
    {\@@@@standaloneframe{#1}[}%
    {\@@@@@@standaloneframe{#1}[}%
}%
\def\@@@@standaloneframe#1[#2]{%
  \@ifnextchar[%]
    {\@@@@@standaloneframe{#1}{#2}}%
    {\begin{frame}#1[#2][environment=standaloneframe]}%
}%
\def\@@@@@standaloneframe#1#2[#3]{%
    \begin{frame}#1[#2][environment=standaloneframe,#3]%
}%
\def\@@@@@@standaloneframe#1[#2]{%
    \begin{frame}#1[environment=standaloneframe,#2]%
}%
\fi
\expandafter\ifx\csname sa@internal@run\endcsname\relax\else
    \AtEndDocument{%
        \immediate\write\@mainaux{\noexpand\@gobbletwo\noexpand\sa@multi@setnumpages{\arabic{sapage}}}%
    }
    \sa@convertfalse
\fi
\ifsa@convert
\ifsa@multi\else
\sa@convert@perpagefalse
\fi
\def\sa@convert@stop{\csname fi\endcsname\stop}%
\begingroup
\let\on@line\@gobble
\def\sa@convert#1{%
    \IfFileExists{\outfile}{%
        \edef\filemodbefore{\csname pdf\ifluatex @\fi filemoddate\endcsname{\outfile}}%
    }{%
    \IfFileExists{\outname\outext}{%
        \edef\filemodbefore{\csname pdf\ifluatex @\fi filemoddate\endcsname{\outname\outext}}%
    }{%
    \IfFileExists{\outname-0\outext}{%
        \edef\filemodbefore{\csname pdf\ifluatex @\fi filemoddate\endcsname{\outname-0\outext}}%
    }{%
    \IfFileExists{\outname-1\outext}{%
        \edef\filemodbefore{\csname pdf\ifluatex @\fi filemoddate\endcsname{\outname-1\outext}}%
    }{%
        \def\filemodbefore{}%
    }}}}%
    \edef\@tempa{\jobname}%
    \edef\@tempb{\sa@convert@subjobname}%
    \@onelevel@sanitize\@tempa
    \@onelevel@sanitize\@tempb
    \@tempswafalse
    \ifx\@tempa\@tempb
        \@tempswatrue
        \edef\infile@filemodbefore{\csname pdf\ifluatex @\fi filemoddate\endcsname{\infile}}%
    \else
        \global\let\sa@convert@stop\relax
    \fi
    \ShellEscape{\sa@convert@latex\space\sa@convert@latexoptions\space
        -jobname \sa@convert@quote\sa@convert@subjobname\sa@convert@quote\space
        \sa@convert@quote\string\expandafter\string\def\string\csname\space
        sa@internal@run\string\endcsname{1}\string\input{\sa@convert@mainfile}\sa@convert@quote}%
    \def\sa@multi@numpages{0}%
    \begingroup
    \IfFileExists{\sa@convert@subjobname.aux}{\@tempswatrue}{\@tempswafalse}%
    \if@tempswa
      \newread\sa@read
      \def\@tempa##1\sa@multi@setnumpages##2##3\@nnil{%
        \def\@tempc{##2}%
        \ifx\@tempc\@nnil\else
          \gdef\sa@multi@numpages{##2}%
        \fi
      }%
      \endlinechar=\m@ne
      \immediate\openin\sa@read=\sa@convert@subjobname.aux\relax
      \loop\unless\ifeof\sa@read
        \read\sa@read to\@tempb
        \expandafter\@tempa\@tempb\sa@multi@setnumpages\@nil\@empty\@nnil
      \repeat
      \immediate\closein\sa@read
    \fi
    \endgroup
    \@tempcnta\sa@multi@numpages\relax
    \ifsa@convert@perpage
    \def\sa@multi@pagemark{-\pagenum}%
    \else
    \ifnum\@tempcnta=\z@
        \def\sa@multi@pagemark{}%
        \edef\sa@lastoutfile{\outfile}%
    \else
        \begingroup
        \def\sa@multi@pagemark{-\the\@tempcnta}%
        \xdef\sa@lastoutfile{\outfile}%
        \endgroup
        \@tempcntb\z@
        \loop\ifnum\@tempcnta>0
            \advance\@tempcntb\@ne
            \divide\@tempcnta by 10\relax
        \repeat
        \edef\sa@multi@pagemark{-\percent0\the\@tempcntb d}%
    \fi
    \fi
    \if@tempswa
        \edef\infile@filemodafter{\csname pdf\ifluatex @\fi filemoddate\endcsname{\infile}}%
        \ifx\infile@filemodbefore\infile@filemodafter
            \global\let\sa@convert@stop\relax
        \fi
    \fi
    \edef\sa@convert@precommand{\sa@convert@precommand}%
    \ifx\sa@convert@precommand\@empty\else
\message{Executing precommand:^^J\sa@convert@precommand^^J^^J}%
        \ShellEscape{\sa@convert@precommand}%
    \fi
    \ifsa@convert@perpage
      \begingroup
      \@tempcnta\sa@multi@numpages\relax
      \@tempcntb\z@
      \def\pagenum{\the\@tempcntb}%
      \loop\ifnum\@tempcntb<\@tempcnta
        \edef\prevpagenum{\pagenum}%
        \advance\@tempcntb\@ne
        \sa@execute@convert@command
        \sa@checkoutfile{\outfile}{#1}%
      \repeat
      \endgroup
    \else
      \sa@execute@convert@command
      \sa@checkoutfile{\sa@lastoutfile}{#1}%
    \fi
    \@tempswafalse
}
\def\sa@execute@convert@command{%
\message{Executing command:^^J\sa@convert@command^^J^^J}%
    \ShellEscape{\sa@convert@command}%
}
\def\sa@checkoutfile#1#2{%
    \IfFileExists{#1}{%
        \edef\filemodafter{\csname pdf\ifluatex @\fi filemoddate\endcsname{#1}}%
        \ifx\filemodbefore\filemodafter
            \expandafter\ifx\csname pdf\ifluatex @\fi filemoddate\endcsname\relax\else
                \sa@convert@failuremsg{standalone}{#2}{}%
            \fi
        \else
            \typeout{Class standalone:^^JOutput written on #1.}%
        \fi
    }{%
        \sa@convert@failuremsg{standalone}{#2}{}%
    }%
}
\let\subjobname\sa@convert@subjobname
\let\mainfile\sa@convert@mainfile
\let\infile\sa@convert@infile
\let\inext\sa@convert@inext
\let\inname\sa@convert@inname
\let\indir\sa@convert@indir
\let\outdir\sa@convert@outdir
\let\outputdir\sa@convert@outputdir
\let\gsdevice\sa@convert@gsdevice
\let\convertexe\sa@convert@convertexe
\let\gsexe\sa@convert@gsexe
\let\inkscapeexe\sa@convert@inkscapeexe
\let\gsexe\sa@convert@gsexe
\let\density\sa@convert@density
\let\size\sa@convert@size
\let\outext\sa@convert@outext
\let\outname\sa@convert@outname
\let\outfile\sa@convert@outfile
\let\pagenum\sa@convert@pagenum
\let\percent\@percentchar
\let\quote\sa@convert@quote
\edef\sa@shellescape{%
    \ifluatex
      \directlua{tex.write(status.shell_escape or 2)}%
    \else\ifxetex
      \the\shellescape
    \else
    \expandafter\ifx\csname pdfshellescape\endcsname\relax
      0%
    \else
      \the\pdfshellescape
    \fi\fi\fi
}%
\ifcase\sa@shellescape\relax% 0
    \sa@convert@failuremsg
        {standalone}{Shell escape disabled! Cannot convert file '\infile'.}{}%
    \global\let\sa@convert@stop\relax
\or% 1
    \sa@convert{Conversion unsuccessful!\MessageBreak
                There might be something wrong with your\MessageBreak
                conversation software or the file permissions!}%
\else% 2 or 3
    \sa@convert{Conversion failed! Please ensure that shell escape\MessageBreak
                is enabled (e.g. use '-shell-escape').}%
\fi
\endgroup
\expandafter\sa@convert@stop
\fi
\begingroup
\toks@\expandafter{%
    \document
    \sa@cls@afterbegindocument
}
\xdef\document{\the\toks@}%
\toks@\expandafter{%
    \expandafter
    \sa@cls@beforeenddocument
    \enddocument
}
\xdef\enddocument{\the\toks@}%
\endgroup
\def\sa@cls@afterbegindocument{\standalone\ignorespaces}
\def\sa@cls@beforeenddocument{\ifhmode\unskip\fi\endstandalone}
\endinput
%%
%% End of file `standalone.cls'.
