###############################################################################

# rpmbuilder:relative-pack true

###############################################################################

%define  debug_package %{nil}

###############################################################################

Summary:         Tiny Redis client for latency measurement
Name:            redis-latency-monitor
Version:         1.0.2
Release:         0%{?dist}
Group:           Applications/System
License:         EKOL
URL:             https://github.com/essentialkaos/redis-latency-monitor

Source0:         https://source.kaos.io/%{name}/%{name}-%{version}.tar.bz2

BuildRoot:       %{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

BuildRequires:   golang >= 1.8

Provides:        %{name} = %{version}-%{release}

###############################################################################

%description
Tiny Redis client for latency measurement.

###############################################################################

%prep
%setup -q

%build
export GOPATH=$(pwd) 
go build src/github.com/essentialkaos/%{name}/%{name}.go

%install
rm -rf %{buildroot}

install -dm 755 %{buildroot}%{_bindir}
install -pm 755 %{name} %{buildroot}%{_bindir}/

%clean
rm -rf %{buildroot}

###############################################################################

%files
%defattr(-,root,root,-)
%doc LICENSE.EN LICENSE.RU
%{_bindir}/%{name}

###############################################################################

%changelog
* Fri Jun 09 2017 Anton Novojilov <andy@essentialkaos.com> - 1.0.2-0
- Improved output

* Thu Jun 08 2017 Anton Novojilov <andy@essentialkaos.com> - 1.0.1-0
- Minor improvements

* Wed Jun 07 2017 Anton Novojilov <andy@essentialkaos.com> - 1.0.0-0
- Initial build
