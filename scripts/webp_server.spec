Name:    webp-server
Version: 0.0.4
Release: 1%{?dist}
Summary: Go version of WebP Server. A tool that will serve your JPG/PNGs as WebP format with compression, on-the-fly.

License: GPLv3
Source0: webp-server
URL: https://github.com/webp-sh/webp_server_go

%description
Go version of WebP Server. A tool that will serve your JPG/PNGs as WebP format with compression, on-the-fly.

%install
%{__mkdir} -p %{buildroot}/%{_bindir}
install -p -m 755 %{SOURCE0} %{buildroot}/%{_bindir}

%files
%{_bindir}/webp-server

%changelog

